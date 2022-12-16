#ifndef QCOM_H_INCLUDED
#define QCOM_H_INCLUDED

#include "QServ.h"

#define ncommand(name, desc, priv, callback, args) { qs.newcommand(name, desc, priv, callback, args); }
#define QSERV_CALLBACK void
#define p int cid, char **args, int argc
#define CMD_NAME(id) qs.getCommandName(id)
#define CMD_DESC(id) qs.getCommandDesc(id)
#define CMD_PRIV(id) qs.getCommandPriv(id)
#define CMD_LAST     qs.getlastCommand()
#define CMD_SENDER   qs.getSender()
#define CMD_SCI      qs.getlastCI()
#define CMD_SA       qs.getlastSA()
#define fulltext     qs.getFullText()

namespace server {
    extern void initCmds();
    extern QSERV_CALLBACK me_cmd(p);
    extern QSERV_CALLBACK stats_cmd(p);
    extern QSERV_CALLBACK localtime_cmd(p);
    extern QSERV_CALLBACK time_cmd(p);
    extern QSERV_CALLBACK bunny_cmd(p);
    extern QSERV_CALLBACK owords_cmd(p);
    extern QSERV_CALLBACK olangfilter_cmd(p);
    extern QSERV_CALLBACK echo_cmd(p);
    extern QSERV_CALLBACK revokepriv_cmd(p);
    extern QSERV_CALLBACK forceintermission_cmd(p);
    extern QSERV_CALLBACK getversion_cmd(p);
    extern QSERV_CALLBACK callops_cmd(p);
    extern QSERV_CALLBACK pm_cmd(p);
    extern QSERV_CALLBACK sendprivs_cmd(p);
    extern QSERV_CALLBACK forgive_cmd(p);
    extern QSERV_CALLBACK forcespectator_cmd(p);
    extern QSERV_CALLBACK unspectate_cmd(p);
    extern QSERV_CALLBACK mute_cmd(p);
    extern QSERV_CALLBACK unmute_cmd(p);
    extern QSERV_CALLBACK editmute_cmd(p);
    extern QSERV_CALLBACK uneditmute_cmd(p);
    extern QSERV_CALLBACK togglelockspec_cmd(p);
    extern QSERV_CALLBACK uptime_cmd(p);
    extern QSERV_CALLBACK whois_cmd(p);
    extern QSERV_CALLBACK help_cmd(p);
    extern QSERV_CALLBACK cheater_cmd(p);
    extern QSERV_CALLBACK mapsucks_cmd(p);
    extern QSERV_CALLBACK ban_cmd(p);
    extern QSERV_CALLBACK pban_cmd(p);
    extern QSERV_CALLBACK clearpbans_cmd(p);
    extern QSERV_CALLBACK teampersist_cmd(p);
    extern QSERV_CALLBACK invadmin_cmd(p);
    extern QSERV_CALLBACK allowmaster_cmd(p);
    extern QSERV_CALLBACK kill_cmd(p);
    extern QSERV_CALLBACK rename_cmd(p);
    extern QSERV_CALLBACK addkey_cmd(p);
    extern QSERV_CALLBACK reloadconfig_cmd(p);
    extern QSERV_CALLBACK listkickbans_cmd(p);
    extern QSERV_CALLBACK unkickban_cmd(p);
    extern QSERV_CALLBACK syncauth_cmd(p);
    extern QSERV_CALLBACK smartbot_cmd(p);
    extern QSERV_CALLBACK cw_cmd(p);
    extern QSERV_CALLBACK duel_cmd(p);
    extern QSERV_CALLBACK coopgamelimit_cmd(p);
    extern QSERV_CALLBACK listmaps_cmd(p);
    extern QSERV_CALLBACK savemap_cmd(p);
    extern QSERV_CALLBACK autosendmap_cmd(p);
    extern QSERV_CALLBACK loadmap_cmd(p);
}

#endif
