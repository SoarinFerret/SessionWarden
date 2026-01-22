#define _GNU_SOURCE
#include <security/pam_modules.h>
#include <security/pam_ext.h>
#include <dbus/dbus.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <syslog.h>
#include <pwd.h>
#include <grp.h>
#include <errno.h>
#include <sys/types.h>
#include <unistd.h>


#define SERVICE_NAME   "io.github.soarinferret.sessionwarden"
#define OBJECT_PATH    "/io/github/soarinferret/sessionwarden"
#define INTERFACE_NAME "io.github.soarinferret.sessionwarden.Manager"

/* List of group names to bypass DBus check for */
static const char *BYPASS_GROUPS[] = { "wheel", "sudo" };
static const int NBYPASS = sizeof(BYPASS_GROUPS)/sizeof(BYPASS_GROUPS[0]);

/* Check if user is in any of groups in BYPASS_GROUPS */
static bool user_in_any_group(const char *username)
{
    struct passwd pwd, *pw = NULL;
    long pw_buf_len = sysconf(_SC_GETPW_R_SIZE_MAX);
    if (pw_buf_len == -1) pw_buf_len = 16384;
    char *pw_buf = malloc(pw_buf_len);
    if (!pw_buf) return false;

    if (getpwnam_r(username, &pwd, pw_buf, pw_buf_len, &pw) != 0 || pw == NULL) {
        free(pw_buf);
        return false;
    }

    /* Resolve target group GIDs */
    gid_t target_gids[NBYPASS];
    for (int i = 0; i < NBYPASS; ++i) {
        struct group grp, *gr = NULL;
        long gr_buf_len = sysconf(_SC_GETGR_R_SIZE_MAX);
        if (gr_buf_len == -1) gr_buf_len = 16384;
        char *gr_buf = malloc(gr_buf_len);
        if (!gr_buf) {
            free(pw_buf);
            return false;
        }
        if (getgrnam_r(BYPASS_GROUPS[i], &grp, gr_buf, gr_buf_len, &gr) == 0 && gr != NULL) {
            target_gids[i] = gr->gr_gid;
        } else {
            /* group not found: set to an impossible GID */
            target_gids[i] = (gid_t)-1;
        }
        free(gr_buf);
    }

    /* Get user's supplementary groups via getgrouplist.
       Start with a reasonable size and grow if needed. */
    int ngroups = 32;
    gid_t *groups = NULL;
    int ret = -1;
    for (;;) {
        free(groups);
        groups = malloc(sizeof(gid_t) * ngroups);
        if (!groups) { free(pw_buf); return false; }
        ret = getgrouplist(username, pw->pw_gid, groups, &ngroups);
        if (ret != -1) break; /* success */
        /* ret == -1, ngroups is set to required size */
        ngroups = (ngroups <= 0) ? 64 : ngroups;
        /* loop and retry with new ngroups */
    }

    /* Now check primary gid and supplementary groups for any target gid */
    bool found = false;
    for (int i = 0; i < NBYPASS && !found; ++i) {
        if (target_gids[i] == (gid_t)-1) continue; /* group doesn't exist on system */
        if (pw->pw_gid == target_gids[i]) { found = true; break; }
        for (int j = 0; j < ngroups; ++j) {
            if (groups[j] == target_gids[i]) { found = true; break; }
        }
    }

    free(groups);
    free(pw_buf);
    return found;
}

// method to write to pam_sessionwarden.log for debugging
static void debug_log(pam_handle_t *pamh, const char *message) {
    FILE *logfile = fopen("/tmp/sessionwarden_pam.log", "a");
    if (logfile) {
        fprintf(logfile, "%s\n", message);
        fclose(logfile);
    }
}

static DBusConnection *connect_system_bus(pam_handle_t *pamh) {
    DBusError err;
    dbus_error_init(&err);

    DBusConnection *conn = dbus_bus_get(DBUS_BUS_SYSTEM, &err);
    if (dbus_error_is_set(&err)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus connection error: %s", err.message);
        debug_log(pamh, "D-Bus connection error");
        dbus_error_free(&err);
        return NULL;
    }
    return conn;
}

static bool check_login_allowed(DBusConnection *conn, pam_handle_t *pamh, const char *user) {
    DBusMessage *msg, *reply;
    DBusMessageIter args;
    DBusError err;
    dbus_error_init(&err);

    msg = dbus_message_new_method_call(
        SERVICE_NAME, OBJECT_PATH,
        INTERFACE_NAME, "CheckLogin"
    );
    if (!msg) {
        pam_syslog(pamh, LOG_ERR, "Failed to create D-Bus message");
        debug_log(pamh, "Failed to create D-Bus message");
        return false;
    }

    // Add username as argument
    dbus_message_append_args(msg, DBUS_TYPE_STRING, &user, DBUS_TYPE_INVALID);

    // Send message and wait for reply
    reply = dbus_connection_send_with_reply_and_block(conn, msg, -1, &err);
    dbus_message_unref(msg);

    if (dbus_error_is_set(&err)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus call failed: %s", err.message);
        // Log detailed error message
        char debug_msg[256];
        snprintf(debug_msg, sizeof(debug_msg), "D-Bus call failed for user %s: %s", user, err.message);
        debug_log(pamh, debug_msg);
        dbus_error_free(&err);
        return false;
    }

    // Extract boolean result
    bool allowed = false;
    if (!dbus_message_iter_init(reply, &args)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus reply has no arguments");
        debug_log(pamh, "D-Bus reply has no arguments");
    } else if (DBUS_TYPE_BOOLEAN != dbus_message_iter_get_arg_type(&args)) {
        pam_syslog(pamh, LOG_ERR, "Unexpected D-Bus reply type");
        debug_log(pamh, "Unexpected D-Bus reply type");
    } else {
        dbus_message_iter_get_basic(&args, &allowed);
    }

    dbus_message_unref(reply);
    return allowed;
}

/* Common implementation for both auth and account management */
static int sessionwarden_check(pam_handle_t *pamh, const char *phase) {
    pam_syslog(pamh, LOG_NOTICE, "sessionwarden called for %s", phase);

    char debug_msg[256];
    snprintf(debug_msg, sizeof(debug_msg), "sessionwarden check called in %s phase", phase);
    debug_log(pamh, debug_msg);

    const char *user = NULL;
    pam_get_user(pamh, &user, NULL);
    if (!user) {
        debug_log(pamh, "No user found");
        return PAM_PERM_DENIED;
    }

    // check if user is in wheel or sudo group
    // if so, allow without checking
    // (this is to allow root and sudo users to always login/unlock)
    if (user_in_any_group(user)) {
        debug_log(pamh, "User is in bypass group, allowing");
        return PAM_SUCCESS;
    }

    // if username is root, allow without checking
    if (strcmp(user, "root") == 0) {
        debug_log(pamh, "User is root, allowing");
        return PAM_SUCCESS;
    }

    DBusConnection *conn = connect_system_bus(pamh);
    if (!conn) {
        debug_log(pamh, "Failed to connect to D-Bus system bus");
        return PAM_PERM_DENIED;
    }

    bool allowed = check_login_allowed(conn, pamh, user);
    if (!allowed) {
        snprintf(debug_msg, sizeof(debug_msg), "Access denied by sessionwarden for %s", user);
        debug_log(pamh, debug_msg);
        pam_syslog(pamh, LOG_NOTICE, "sessionwarden denied access for %s", user);
        return PAM_PERM_DENIED;
    }

    snprintf(debug_msg, sizeof(debug_msg), "Access allowed by sessionwarden for %s", user);
    debug_log(pamh, debug_msg);

    return PAM_SUCCESS;
}

PAM_EXTERN int pam_sm_authenticate(pam_handle_t *pamh, int flags,
                                   int argc, const char **argv) {
    // Called during auth phase (login AND screen unlock)
    // Does NOT verify password - that's pam_unix.so's job
    // Only checks if user should be allowed based on time limits
    return sessionwarden_check(pamh, "authentication");
}

PAM_EXTERN int pam_sm_setcred(pam_handle_t *pamh, int flags,
                              int argc, const char **argv) {
    // Required when implementing pam_sm_authenticate
    // We don't actually need to set any credentials, just return success
    return PAM_SUCCESS;
}

PAM_EXTERN int pam_sm_acct_mgmt(pam_handle_t *pamh, int flags,
                                   int argc, const char **argv) {
    // Called during account management phase (login only, not unlock)
    return sessionwarden_check(pamh, "account management");
}
