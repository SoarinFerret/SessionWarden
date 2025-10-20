#include <security/pam_modules.h>
#include <security/pam_ext.h>
#include <dbus/dbus.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <syslog.h>

#define SERVICE_NAME   "io.github.yourname.sessionwarden"
#define OBJECT_PATH    "/io/github/yourname/sessionwarden"
#define INTERFACE_NAME "io.github.yourname.sessionwarden.Manager"

static DBusConnection *connect_system_bus(pam_handle_t *pamh) {
    DBusError err;
    dbus_error_init(&err);

    DBusConnection *conn = dbus_bus_get(DBUS_BUS_SYSTEM, &err);
    if (dbus_error_is_set(&err)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus connection error: %s", err.message);
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
        return false;
    }

    // Add username as argument
    dbus_message_append_args(msg, DBUS_TYPE_STRING, &user, DBUS_TYPE_INVALID);

    // Send message and wait for reply
    reply = dbus_connection_send_with_reply_and_block(conn, msg, -1, &err);
    dbus_message_unref(msg);

    if (dbus_error_is_set(&err)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus call failed: %s", err.message);
        dbus_error_free(&err);
        return false;
    }

    // Extract boolean result
    bool allowed = false;
    if (!dbus_message_iter_init(reply, &args)) {
        pam_syslog(pamh, LOG_ERR, "D-Bus reply has no arguments");
    } else if (DBUS_TYPE_BOOLEAN != dbus_message_iter_get_arg_type(&args)) {
        pam_syslog(pamh, LOG_ERR, "Unexpected D-Bus reply type");
    } else {
        dbus_message_iter_get_basic(&args, &allowed);
    }

    dbus_message_unref(reply);
    return allowed;
}

static void notify_logout(DBusConnection *conn, pam_handle_t *pamh, const char *user) {
    DBusMessage *msg;
    msg = dbus_message_new_method_call(
        SERVICE_NAME, OBJECT_PATH,
        INTERFACE_NAME, "NotifyLogout"
    );
    if (!msg) {
        pam_syslog(pamh, LOG_ERR, "Failed to create logout D-Bus message");
        return;
    }

    dbus_message_append_args(msg, DBUS_TYPE_STRING, &user, DBUS_TYPE_INVALID);

    if (!dbus_connection_send(conn, msg, NULL)) {
        pam_syslog(pamh, LOG_ERR, "Failed to send logout D-Bus message");
    }

    dbus_message_unref(msg);
}

PAM_EXTERN int pam_sm_open_session(pam_handle_t *pamh, int flags,
                                   int argc, const char **argv) {
    const char *user = NULL;
    pam_get_user(pamh, &user, NULL);
    if (!user) return PAM_PERM_DENIED;

    DBusConnection *conn = connect_system_bus(pamh);
    if (!conn) return PAM_PERM_DENIED;

    bool allowed = check_login_allowed(conn, pamh, user);
    if (!allowed) {
        pam_syslog(pamh, LOG_NOTICE, "sessionwarden denied login for %s", user);
        return PAM_PERM_DENIED;
    }

    return PAM_SUCCESS;
}

PAM_EXTERN int pam_sm_close_session(pam_handle_t *pamh, int flags,
                                    int argc, const char **argv) {
    const char *user = NULL;
    pam_get_user(pamh, &user, NULL);
    if (!user) return PAM_SUCCESS;

    DBusConnection *conn = connect_system_bus(pamh);
    if (!conn) return PAM_SUCCESS;

    notify_logout(conn, pamh, user);
    return PAM_SUCCESS;
}
