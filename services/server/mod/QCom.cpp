#include "QCom.h"
#include "QServ.h"

namespace server {
    void initCmds() {
        /**
            @ncommand = New Command
            @Command name
            @Command description and usage
            @Command callback function
            @Command argument count
        **/
        ncommand("help", "\f7View command list or command usage. \nUsage: #help for command list and #help <name-of-command> for usage", PRIV_NONE, help_cmd, 1);
        ncommand("me", "\f7Echo your name and message to everyone. Usage: #me <message>", PRIV_NONE, me_cmd, 0);
        ncommand("stats", "\f7View the stats of a player or yourself. Usage: #stats <cn> or #stats", PRIV_NONE, stats_cmd, 1);
        ncommand("localtime", "\f7Get the local time of the server. Usage: #localtime", PRIV_ADMIN, localtime_cmd, 0);
        ncommand("ver", "\f7Get the current QServ version. Usage: #ver", PRIV_NONE, getversion_cmd, 0);
        ncommand("uptime", "\f7View how long the server has been up for. Usage: #uptime", PRIV_NONE, uptime_cmd, 0);
        ncommand("invadmin", "\f7Claim invisible administrator. Usage: #invadmin <adminpass>", PRIV_NONE, invadmin_cmd, 1);
        ncommand("cheater", "\f7Accuses someone of cheating and alerts moderators. Usage: #cheater <cn>", PRIV_NONE, cheater_cmd, 1);
        ncommand("whois", "\f7View information about a player. Usage: #whois <cn>", PRIV_NONE, whois_cmd, 1);
        ncommand("time", "\f7View the current time. Usage: #time <UTC Offset Number>", PRIV_MASTER, time_cmd, 1);
        ncommand("pm", "\f7Send a private message to someone. Usage #pm <cn> <private message>", PRIV_NONE, pm_cmd,2);
        ncommand("callops", "\f7Call all operators on the Internet Relay Chat Server. Usage: #callops", PRIV_NONE, callops_cmd, 0);
        ncommand("mapsucks", "\f7Votes for an intermission to change the map. Usage: #mapsucks", PRIV_NONE, mapsucks_cmd, 0);
        ncommand("forgive", "\f7Forgive a player for teamkilling or just in general. Usage: #forgive <cn>", PRIV_NONE, forgive_cmd, 1);
        ncommand("intermission", "\f7Force an intermission. Usage: #intermission", PRIV_MASTER, forceintermission_cmd, 0);
        ncommand("echo", "\f7Broadcast a message to all players. Usage: #echo <message>", PRIV_MASTER, echo_cmd, 1);
        ncommand("sendprivs", "\f7Share master/admin with another player. Usage: #sendprivs <cn>", PRIV_MASTER, sendprivs_cmd, 1);
        ncommand("bunny", "\f7Broadcast a helper message to all players. Usage: #bunny <helpmessage>", PRIV_ADMIN, bunny_cmd, 0);
        ncommand("revokepriv", "\f7Revoke the privileges of a player. Usage: #revokepriv <cn>", PRIV_ADMIN, revokepriv_cmd, 1);
        ncommand("forcespectator", "\f7Forces a player to become a spectator. Usage: #forcespectator <cn>", PRIV_ADMIN, forcespectator_cmd, 1);
        ncommand("unspectate", "\f7Removes a player from spectator mode. Usage: #unspectate <cn>", PRIV_ADMIN, unspectate_cmd, 1);
        ncommand("mute", "\f7Mutes a client. Usage #mute <cn>", PRIV_ADMIN, mute_cmd, 1);
        ncommand("unmute", "\f7Unmutes a client. Usage #mute <cn>", PRIV_ADMIN, unmute_cmd, 1);
        ncommand("editmute", "\f7Stops a client from editing. Usage #editmute <cn>", PRIV_ADMIN, editmute_cmd, 1);
        ncommand("uneditmute", "\f7Allows a client to edit again. Usage #uneditmute <cn>", PRIV_ADMIN, uneditmute_cmd, 1);
        ncommand("togglelockspec", "\f7Forces a client to be locked in spectator mode. Usage #togglelockspec <cn>", PRIV_ADMIN, togglelockspec_cmd, 1);
        ncommand("ban", "\f7Bans a client. Usage: #ban <cn> <ban time in minutes>", PRIV_ADMIN, ban_cmd, 2);
        ncommand("pban", "\f7Permanently bans a client. Not listed on #listkickbans, not undoable. Use #clearpbans to clear all. Usage: #pban <cn>", PRIV_ADMIN, pban_cmd, 1);
        ncommand("clearpbans", "\f7Clears all pbans and ipbans. Usage: #clearpbans", PRIV_ADMIN, clearpbans_cmd, 0);
        ncommand("teampersist", "\f7Toggle persistant teams on or off. Usage: #teampersist <0/1> (0 for off, 1 for on)", PRIV_MASTER, teampersist_cmd, 1);
        ncommand("allowmaster", "\f7Allows clients to claim master. Usage: #allowmaster <0/1> (0 for off, 1 for on)", PRIV_ADMIN, allowmaster_cmd, 1);
        ncommand("kill", "\f7Brutally murders a player. Usage: #kill <cn>", PRIV_ADMIN, kill_cmd, 1);
        ncommand("rename", "\f7Renames a player. Usage: #rename <cn> <new name>", PRIV_ADMIN, rename_cmd, 2);
        ncommand("addkey", "\f7Adds an authkey to the server. \nUsage: #addkey <name> <domain> <public key> <privilege>", PRIV_ADMIN, addkey_cmd, 4);
        ncommand("listkickbans", "\f7Lists all kicks/bans. Usage: #listkickbans", PRIV_ADMIN, listkickbans_cmd, 0);
        ncommand("reloadconfig","\f7Reloads server-init.cfg configuration. Usage: #reloadconfig", PRIV_ADMIN, reloadconfig_cmd, 0);
        ncommand("unkickban", "\f7Unkick/unbans a player. Usage: #unban <ID>. Use #listkickbans for a list with ID's", PRIV_ADMIN, unkickban_cmd, 1);
        ncommand("syncauth", "\f7Sync server with new authkeys added to users.cfg. Usage: #syncauth", PRIV_ADMIN, syncauth_cmd, 0);
        ncommand("cw", "\f7Starts a clanwar with a countdown (timer dependent on maxclients). Usage: #cw <mode> <map>", PRIV_MASTER, cw_cmd, 2);
        ncommand("duel", "\f7Starts a duel (timer dependent on maxclients). Usage: #duel <mode> <map>", PRIV_MASTER, duel_cmd, 2);
        ncommand("icgl", "\f7Sets the game limit in milliseconds for instacoop. Usage: #icgl <limit in milliseconds>", PRIV_ADMIN, coopgamelimit_cmd, 1);
        ncommand("listmaps", "\f7Lists all the maps stored on the server. Usage #listmaps", PRIV_ADMIN, listmaps_cmd, 0);
        ncommand("savemap", "\f7Saves a map to the server. Usage #savemap", PRIV_ADMIN, savemap_cmd, 0);
        ncommand("autosendmap", "\f7Automatically sends the map to connecting clients. Usage #autosendmap <1/0> (0 for off, 1 for on)", PRIV_MASTER, autosendmap_cmd, 1);
        ncommand("loadmap", "\f7Loads a map stored on the server. Usage #loadmap <mapname>", PRIV_ADMIN, loadmap_cmd, 1);
        //ncommand("smartbot", "\f7Speak to smart IRC bot. Usage: #smartbot <command> \nUse: #smartbot help for a list of commands", PRIV_NONE, smartbot_cmd, 1);
        //ncommand("owords", "View list of offensive words. Usage: #owords",PRIV_NONE, owords_cmd, 0);
        //ncommand("olangfilter", "Turn the offensive language filter on or off. Usage: #olang <off/on> (0/1) and #olang to see if it's activated", PRIV_MASTER, olangfilter_cmd, 1);
    }
    
    QSERV_CALLBACK loadmap_cmd(p) {
        if(strlen(fulltext) > 0) {
            server::loadmap(fulltext);
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
    
    QSERV_CALLBACK autosendmap_cmd(p) {
        bool usage = false;
        int togglenum = -1;
        if(CMD_SA) {
            togglenum = atoi(args[1]);
            autosendmapprocess:
                if(togglenum==1 && enableautosendmap == false) {
                    server::enableautosendmap = true;
                    clientinfo *ci = qs.getClient(CMD_SENDER);
                    out(ECHO_SERV, "\f7Autosendmap is now \f0enabled");
                }
                else if(togglenum==0 && enableautosendmap == true) {
                    server::enableautosendmap = false;
                    out(ECHO_SERV, "\f7Autosendmap is now \f3disabled");
                }
                else if(togglenum==0 && enableautosendmap == false) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Autosendmap is already disabled. Use \f2#autosendmap 1 \f3to enable it.");
                else if(togglenum==1 && enableautosendmap == true) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Autosendmap is already enabled. Use \f2#autosendmap 0 \f3to disable it.");
                else if(togglenum==NULL || isalpha(togglenum) || togglenum < 0 || togglenum > 1) usage = true;
        } else {
            togglenum = -1;
            usage = true;
        }
        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK savemap_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "Saved map to server. Use #listmaps to see all server-stored maps");
        server::dosavemap();
    }
    
    QSERV_CALLBACK listmaps_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        server::listmaps(ci->clientnum);
    }
    
    extern int instacoop_gamelimit;
     QSERV_CALLBACK coopgamelimit_cmd(p) {
        int Limitvariable = -1;
        if(CMD_SA) {
        	Limitvariable = atoi(args[1]);
        	if(Limitvariable < 1000 || Limitvariable > 9999999) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Enter a game limit between the range of 1000 to 9999999 milliseconds.");
        	else if(Limitvariable != NULL && args[1] != NULL) instacoop_gamelimit = Limitvariable;
        }
        else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));

    }
    
    QSERV_CALLBACK smartbot_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        if(strlen(fulltext) > 0) {
            out(ECHO_NOCOLOR,".%s", fulltext);
            out(ECHO_SERV,"\f7%s: \f0#smartbot %s", colorname(ci), fulltext); //format command for server
        } else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK reloadconfig_cmd(p) {
        execfile("config/modifier.cfg", false);
        out(ECHO_CONSOLE, "Reloaded configuration from config/modifier.cfg");
        out(ECHO_ALL, "Server has reloaded configuration. Update your server list and reconnect to see changes");
    }
    QSERV_CALLBACK syncauth_cmd(p) {
        execfile("config/users.cfg", false);
        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "Resynced auth. All keys added to config/users.cfg will now be active.");
        out(ECHO_ALL, "Server has reloaded authkeys. New keys are now active.");
    }
    
    QSERV_CALLBACK addkey_cmd(p) {
        if(CMD_SA && args[1] != NULL && args[2] != NULL && args[3] != NULL && args[4] != NULL) {
            server::adduser(args[1],args[2],args[3],args[4]);
            FILE * userscfg;
            userscfg = fopen ( "config/users.cfg" , "a"); //append mode
            defformatstring(fullkey)("\nadduser %s %s %s %s\n", args[1],args[2],args[3],args[4]);
            defformatstring(authmsg)("Added key: %s %s %s %s",args[1],args[2],args[3],args[4]);
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, authmsg);
            fputs (fullkey, userscfg);
            fclose (userscfg);
        }
        else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK listkickbans_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        server::sendkickbanlist(ci->clientnum);
    }
    
    QSERV_CALLBACK unkickban_cmd(p) {
        if(CMD_SA){
        	clientinfo *ci = qs.getClient(CMD_SENDER);
            int banid = atoi(args[1]);
            server::unkickban(banid, ci->clientnum);
        }
        else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK rename_cmd(p) {
        if(CMD_SA) {
            int cn = atoi(args[1]);
            char *name = args[2];
            clientinfo *ci = qs.getClient(cn);
            if(!ci || name == NULL || name[0] == 1 || cn < 0 || cn >= 128) return;
            char newname[MAXNAMELEN + 1];
            newname[MAXNAMELEN] = 0;
            filtertext(newname, name, false, MAXNAMELEN);
            putuint(ci->messages, N_SWITCHNAME);
            sendstring(newname, ci->messages);
            vector<uchar> buf;
            putuint(buf, N_SWITCHNAME);
            sendstring(newname, buf);
            packetbuf v(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
            putuint(v, N_CLIENT);
            putint(v, ci->clientnum);
            putint(v, buf.length());
            v.put(buf.getbuf(), buf.length());
            sendpacket(ci->clientnum, 1, v.finalize(), -1);
            string oldname;
            copystring(oldname, ci->name);
            copystring(ci->name, newname);
        }
        else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK teampersist_cmd(p) {
        bool usage = false;
        int togglenum = -1;
        if(CMD_SA) {
            togglenum = atoi(args[1]);
                if(togglenum==1 && server::persist == false) {
                    server::persist = true;
                    clientinfo *ci = qs.getClient(CMD_SENDER);
                    out(ECHO_SERV, "\f7Persistant teams are now \f0enabled");
                }
                else if(togglenum==0 && server::persist == true) {
                    server::persist = false;
                    out(ECHO_SERV, "\f7Persistant teams are now \f3disabled");
                }
                else if(togglenum==0 && server::persist == false) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Persistant teams are already disabled. Use \f2#teampersist 1 \f3to enable them.");
                else if(togglenum==1 && server::persist == true) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Persistant teams are already enabled. Use \f2#teampersist 0 \f3to disable them.");
                else if(togglenum==NULL || isalpha(togglenum) || togglenum < 0 || togglenum > 1) usage = true;
        } else {
            togglenum = -1;
            usage = true;
        }
        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
     extern void suicide(clientinfo *ci);
     QSERV_CALLBACK kill_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    killprocess:
                    clientinfo *ci = qs.getClient(cn);
                    if(ci != NULL) {
                        if(ci->connected) {
                         	if(cn!=CMD_SENDER && !isalpha(cn) && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL && cn!=CMD_SENDER) {
            				clientinfo *ci = qs.getClient(cn);
            				clientinfo *sender = qs.getClient(CMD_SENDER);
           		 				if(ci->state.state==CS_ALIVE) {
            						suicide(ci);
            						out(ECHO_SERV, "\f0%s \f7has been brutally murdered", colorname(ci));
            						out(ECHO_NOCOLOR, "%s has been brutally murdered by %s", colorname(ci), colorname(sender));
            					}   
                        	}
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto killprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
	QSERV_CALLBACK allowmaster_cmd(p) {
		bool usage = false;
	    int togglenum = -1;
	    if(CMD_SA) {
	        togglenum = atoi(args[1]);
	        	if(togglenum==1 && mastermask == MM_PUBSERV) {
	            	switchallowmaster();
	            	clientinfo *ci = qs.getClient(CMD_SENDER);
	            	out(ECHO_SERV, "\f7Claiming \f0master \f7with \"/setmaster 1\" is now \f0enabled");
	        	}
	        	else if(togglenum==0 && mastermask == MM_PRIVSERV) {
	            	switchdisallowmaster();
	            	out(ECHO_SERV, "\f7Claiming \f0master \f7with \"/setmaster 1\" is now \f3disabled");
	        	}
	        	else if(togglenum==0 && mastermask == MM_PUBSERV) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Master is already disabled. Use \f2#allowmaster 1 \f3to enable it.");
                else if(togglenum==1 && mastermask == MM_PRIVSERV) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Master is already enabled. Use \f2#allowmaster 0 \f3to disable it.");
                else if(togglenum==NULL || isalpha(togglenum) || togglenum < 0 || togglenum > 1) usage = true;
	    } else {
	        togglenum = -1;
	        usage = true;
	    }
		if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
	}
    
    SVAR(invadminpass, "");
    QSERV_CALLBACK invadmin_cmd(p) {
        if(CMD_SA) {
            clientinfo *ci = qs.getClient(CMD_SENDER);
            if(!strcmp(invadminpass, args[1]) || invadminpass == args[1]) {
                ci->privilege = PRIV_ADMIN;
                ci->isInvAdmin = true;
                sendf(ci->clientnum, 1, "ris", N_SERVMSG, "Invisible \f6admin \f7activated");
                out(ECHO_IRC, "%s became an invisible admin", colorname(ci));
                out(ECHO_CONSOLE, "%s became an invisible admin", colorname(ci));
            }
            else sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Incorrect admin password");
        }
    }
        
    QSERV_CALLBACK clearpbans_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        sendf(ci->clientnum, 1, "ris", N_SERVMSG, "Cleared all IP bans");
        server::clearpbans();
    }
    
    QSERV_CALLBACK ban_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                clientinfo *ci = qs.getClient(cn);
                if(ci != NULL) {
                    if(ci->connected) {
                        if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL && cn!=CMD_SENDER) {
                            if(args[2] != NULL) { 
                                uint ip = getclientip(ci->clientnum);
                                int expiremilliseconds = atoi(args[2])*60000;    
                                int expireseconds = (expiremilliseconds/1000);   
                                int expireminutes = (expireseconds/60);          
                                if(expireminutes < 60 && expireminutes > 0) {
                                	addban(ip, expiremilliseconds);
                                	clientinfo *sender = qs.getClient(CMD_SENDER);
                                	disconnect_client(cn, DISC_KICK);
                                	out(ECHO_SERV, "\f0%s \f7has been banned for %d minutes.", colorname(ci), expireminutes);
                                	out(ECHO_NOCOLOR, "%s has been banned for %d minutes.", colorname(ci), expireminutes);
                                }
                                else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Ban time must be between 1 and 59 minutes.");
                            }
                            else if(args[2] == NULL) usage = true;
                        }
                    }
                } else {
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                }
                
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            usage = true;
        }
        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK pban_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                clientinfo *ci = qs.getClient(cn);
                if(ci != NULL) {
                    if(ci->connected) {
                        if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL && cn!=CMD_SENDER) {
                            //ipban doesn't get listed on listkickbans
                            clientinfo *ci = qs.getClient(cn);
                            out(ECHO_SERV, "\f0%s \f7has been permanently banned.", colorname(ci));
                            out(ECHO_NOCOLOR, "%s has been permanently banned.", colorname(ci));
                            server::ipban(ci->ip);
                            disconnect_client(cn, DISC_IPBAN);
                        }
                    }
                } else {
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                }
                
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            usage = true;
        }
        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    VAR(votestopassmapsucks, 2, 10, INT_MAX);
    int mapsucksvotes = 0;
    QSERV_CALLBACK mapsucks_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        if(!ci->votedmapsucks) {
            mapsucksvotes++;
            out(ECHO_SERV, "\f0%s \f7thinks this map sucks, use \f2#mapsucks \f7to vote for an intermission to skip it.", colorname(ci));
            ci->votedmapsucks = true;
            if(mapsucksvotes>=getvar("votestopassmapsucks")) {
                startintermission();
                out(ECHO_SERV, "\f7Changing map: That map sucked (\f7%d \f7votes to skip)", votestopassmapsucks);
                while(mapsucksvotes>=getvar("votestopassmapsucks")) {
                    mapsucksvotes = 0;
                    out(ECHO_SERV,"\f7Clearing votes...");
                }
            }
        }
        else if(ci->votedmapsucks) sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: You have already voted");
    }
    
    //min, default, max
    VAR(clanwartimermillis, 5000, 8999, 10000);
    int mc = 22;
    extern void changemap(const char *s, int mode);
    extern void pausegame(bool val, clientinfo *ci = NULL);
    QSERV_CALLBACK cw_cmd(p) {
        const char *mapname = args[2];
        char *mn = args[1];
        if(CMD_SA && args[1] != NULL && args[2] != NULL && *mapname !=NULL && *mn!=NULL) {
        int gm; // default set to current mode td
        bool valid = false;
        
        // intialize this for perf
        for(int i = 0; i <= mc; i++)  {
            if(!strcmp(mn, qserv_modenames[i]))  {
                gm = i;
                // use other list to send full name of mode
                changemap(mapname, gm);
                valid = true;
                break;
            }
        }
        if(CMD_SA && valid) {
            clientinfo *ci = qs.getClient(CMD_SENDER);
            server::persist = true;
            pausegame(true,ci);
            int cwtimer;
            for(cwtimer = clanwartimermillis; cwtimer <= clanwartimermillis && cwtimer != (0); --cwtimer) {
                out(ECHO_SERV, "\f2Get ready! Starting clanwar in %d:%d",(cwtimer/1000) % 1000,cwtimer);
            }
            pausegame(false,ci);
            defformatstring(f)("\f2Clanwar has started: %s on map %s. Good luck, have fun.", qserv_modenames[gm], mapname);
            sendf(-1, 1, "ris", N_SERVMSG, f);
        }
            
        if(!valid) {sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Unknown mode");}
        }
        else {
        	bool valid = false;
            defformatstring(f)("\f3Error: Invalid mode/mapname provided");
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, f);
        }
    }
    
    //min, default, max
    VAR(dueltimermillis, 5000, 8999, 10000);
    int mcduel = 22;
    QSERV_CALLBACK duel_cmd(p) {
        const char *mapname = args[2];
        char *mn = args[1];
        if(CMD_SA && args[1] != NULL && args[2] != NULL && *mapname !=NULL && *mn!=NULL) {
            int gm; // default set to current mode td
            bool valid = false;
            
            // intialize this for perf
            for(int i = 0; i <= mcduel; i++)  {
                if(!strcmp(mn, qserv_modenames[i]))  {
                    gm = i;
                    // use other list to send full name of mode
                    changemap(mapname, gm);
                    valid = true;
                    break;
                }
            }
            if(CMD_SA && valid) {
                clientinfo *ci = qs.getClient(CMD_SENDER);
                server::persist = true;
                pausegame(true,ci);
                int dueltimer;
                for(dueltimer = dueltimermillis; dueltimer <= dueltimermillis && dueltimer != (0); --dueltimer) {
                    out(ECHO_SERV, "\f2Get ready! Starting duel in %d:%d",(dueltimer/1000) % 1000,dueltimer);
                }
                pausegame(false,ci);
                defformatstring(f)("\f2Duel has started: %s on map %s. Good luck, have fun.", qserv_modenames[gm], mapname);
                sendf(-1, 1, "ris", N_SERVMSG, f);
            }
            
            if(!valid) {sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Unknown mode");}
        }
        else {
            bool valid = false;
            defformatstring(f)("\f3Error: Invalid mode/mapname provided");
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, f);
        }
    }
    
    QSERV_CALLBACK uptime_cmd(p) {
        clientinfo *ci = qs.getClient(CMD_SENDER);
        string msg, buf;
        uint t, months, weeks, days, hours, minutes, seconds;
        copystring(msg,"\f7Server Mod: \f3QServ\f7: \f1https://github.com/deathstar/QServCollect");
        sendf(ci ? ci->clientnum : -1, 1, "ris", N_SERVMSG, msg);
        copystring(msg, "\f7Server Architecture: \f0"
		/* Firstly determine OS */
		#if !(defined(_WIN32) || defined(WIN32) || defined(WIN64) || defined(_WIN64))
		/* unix/posix compilant os */
		#   if defined(__linux__) || defined(__linux) || defined(linux) || defined(__gnu_linux__)
                   "GNU/Linux"
		#   elif defined(__GNU__) || defined(__gnu_hurd__)
                   "GNU/Hurd"
		#   elif defined(__FreeBSD_kernel__) && defined(__GLIBC__)
                   "GNU/FreeBSD"
		#   elif defined(__FreeBSD__) || defined(__FreeBSD_kernel__)
                   "FreeBSD"
		#   elif defined(__OpenBSD__)
                   "OpenBSD"
		#   elif defined(__NetBSD__)
                   "NetBSD"
		#   elif defined(__sun) || defined(sun)
                   "Solaris"
		#   elif defined(__DragonFly__)
                   "DragonFlyBSD"
		#   elif defined(__MACH__)
		#       if defined(__APPLE__)      
                   "Apple"     
		#       else   
                   "Mach"  
		#       endif  
		#   elif defined(__CYGWIN__)  
                   "Cygwin"  
		#   elif defined(__unix__) || defined(__unix) || defined(unix) || defined(_POSIX_VERSION)  
                   "UNIX"  
		#   else  
                   "unknown"   
		#   endif     
		#else      
                   /* Windows */
                   "Windows"
		#endif
                   " "
                   );
        concatstring(msg, (sizeof(void *) == 8) ? "x86 (64 bit)" : "i386");
        concatstring(msg, "\n\f7Server Uptime:\f6");
        t = totalsecs;
        
        months = t / (30*24*60*60);
        
        t = t % (30*24*60*60);
        
        weeks = t / (7*24*60*60);
        
        t = t % (7*24*60*60);
        
        days = t / (24*60*60);
        
        t = t % (24*60*60);
        
        hours = t / (60*60);
        
        t = t % (60*60);
        
        minutes = t / 60;
        
        t = t % 60;
        
        seconds = t;
    
        if(months)    
        {
            formatstring(buf)(" %u month%s", months, months > 1 ? "s" : "");
            concatstring(msg, buf);
        }
        
        if(weeks)
        { 
            formatstring(buf)(" %u week%s", weeks, weeks > 1 ? "s" : "");
            concatstring(msg, buf);  
        }

        if(days)
        {
            formatstring(buf)(" %u day%s", days, days > 1 ? "s" : "");
            concatstring(msg, buf);
        }
        
        if(hours)  
        { 
            formatstring(buf)(" %u hour%s", hours, hours > 1 ? "s" : "");
            concatstring(msg, buf);  
        }
          
        if(minutes)   
        {
            formatstring(buf)(" %u minute%s", minutes, minutes > 1 ? "s" : "");
            concatstring(msg, buf);  
        }
        
        if(seconds)
        {
            formatstring(buf)(" %u second%s", seconds, seconds > 1 ? "s" : "");
            concatstring(msg, buf);    
        }
        sendf(ci ? ci->clientnum : -1, 1, "ris", N_SERVMSG, msg);
    }

     QSERV_CALLBACK whois_cmd(p) {
         #include <stdio.h>
         //check to see if we want to use curl
         FILE* f_mode = fopen("usecurl.txt", "r");
         bool curlgeolocation;
         if(f_mode) {curlgeolocation = true;}
         else {curlgeolocation = false;}
         
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    sendinfo:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                            char *ip = toip(cn), lmsg[3][255];
                            const char *location;

                            if(!strcmp("127.0.0.1", ip)) {
                                location = (char*)"localhost";
                            } else {
                                location = qs.cgip(ip).c_str(); //qs.congeoip(ip); <-- just country

                                if(!location) {
                                    sprintf(lmsg[1], "%s", "Unknown Location");
                                }
                            }
                            if(location) sprintf(lmsg[1], "%s", location);
                            (CMD_SCI.privilege == PRIV_ADMIN) ? sprintf(lmsg[0], "%s (%s)", lmsg[1], ip) : sprintf(lmsg[0], "%s", lmsg[1]);
                            if(curlgeolocation) {
                                defformatstring(s)("Name: \f0%s \f7CN: \f1%d", colorname(ci), ci->clientnum);
                                sendf(CMD_SENDER, 1, "ris", N_SERVMSG, s);
                                
                            }
                            else {
                                 defformatstring(s)("Name: \f0%s \f7CN: \f1%d \f7Location: \f2%s",colorname(ci), ci->clientnum,lmsg[0]);
                                sendf(CMD_SENDER, 1, "ris", N_SERVMSG, s);
                            }
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto sendinfo;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    extern void forcespectator(clientinfo *ci);
    extern void unspectate(clientinfo *ci);
     QSERV_CALLBACK togglelockspec_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    lockspecprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        
                        	if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
            					if(!ci->isSpecLocked) {
                					forcespectator(ci);
                					ci->isSpecLocked = true;
                					sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3You have been locked in spectator mode.");
            				} else if(ci->isSpecLocked) {
                					unspectate(ci);
                					ci->isSpecLocked = false;
                					sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3You are no longer locked in spectator mode.");
            			}
            
        			}
                           
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto lockspecprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
      QSERV_CALLBACK forcespectator_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    forcespectatorprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                                if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
        						clientinfo *ci = qs.getClient(cn);
        						forcespectator(ci);
        				}				

                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto forcespectatorprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK unspectate_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    unspectateprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        	if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
        						clientinfo *ci = qs.getClient(cn);
       						    unspectate(ci);
        				}

                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto unspectateprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK mute_cmd(p) {
        bool usage = false;
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    muteprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        	if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
            					ci->isMuted = true;
            					defformatstring(mutemsg)("\f0%s \f7has been \f3muted", colorname(ci));
            					sendf(CMD_SENDER, 1, "ris", N_SERVMSG, mutemsg);
        					}
                        }
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto muteprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
     QSERV_CALLBACK unmute_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    unmuteprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        	if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
            					ci->isMuted = false;
            					defformatstring(unmutemsg)("\f0%s \f7has been \f0unmuted", colorname(ci));
           						 sendf(CMD_SENDER, 1, "ris", N_SERVMSG, unmutemsg);
        					}
							
                       }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto unmuteprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }

	QSERV_CALLBACK editmute_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    editmuteprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        	 if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
            					ci->isEditMuted = true;
            					sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f7Your edits \f3will not \f7show up to others.");
           					    defformatstring(mutemsg)("\f0%s\f7's edits have been \f3muted", colorname(ci));
            					sendf(CMD_SENDER, 1, "ris", N_SERVMSG, mutemsg);
       						 }
                                                   
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto editmuteprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK uneditmute_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    uneditmuteprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        
                        	if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
            					sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f7Your edits \f0will \f7now show up to others.");
            					ci->isEditMuted = false;
            					out(ECHO_SERV,"\f0%s\f7's edits have been \f0unmuted", colorname(ci));
        					}                         
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto uneditmuteprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    QSERV_CALLBACK forgive_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    forgiveprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        	clientinfo *self = qs.getClient(CMD_SENDER);
        					if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && ci != NULL && ci->connected && args[1] != NULL) {
                                if(ci->state.teamkills >= 1) {
                                	defformatstring(forgivemsg)("\f0%s \f7has forgiven \f3%s", colorname(self), colorname(ci));
                                	sendf(-1, 1, "ris", N_SERVMSG, forgivemsg);
                                }
                                else {
                                	defformatstring(nk)("\f3Error: %s has not teamkilled anyone yet", colorname(ci));
                                	sendf(-1, 1, "ris", N_SERVMSG, nk);
                                }  
                            }
                            
                       }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto forgiveprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
    
    SVAR(ircoperators, "");
     QSERV_CALLBACK cheater_cmd(p) {
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    cheaterprocess:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                        
                        clientinfo *self = qs.getClient(CMD_SENDER);
        				if(cn!=CMD_SENDER && cn >= 0 && cn <= 1000 && cn != NULL && ci != NULL && ci->connected && args[1]!=NULL) {
        				int accuracy = (ci->state.damage*100)/max(ci->state.shotdamage, 1);
            			privilegemsg(PRIV_MASTER,"\f7Something's fishy! A cheater has been reported.");
             			out(ECHO_SERV, "\f0%s \f7accuses \f3%s \f7(CN: \f6%d \f7| Accuracy: \f6%d%\f7) of cheating.", colorname(self), colorname(ci), ci->clientnum, accuracy);
            			out(ECHO_NOCOLOR, "Attention Operator(s): %s - %s accuses %s (CN: %d | Accuracy: %d%) of cheating.", ircoperators, colorname(self), colorname(ci), ci->clientnum, accuracy);
            			defformatstring(nocolorcheatermsg)("\f3%s \f7has been reported.", colorname(ci));
            			sendf(CMD_SENDER, 1, "ris", N_SERVMSG, nocolorcheatermsg);
        			}		
                           
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto cheaterprocess;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
	
    QSERV_CALLBACK olangfilter_cmd(p) {
        if(CMD_SA) {
            int state = atoi(args[1]);

            if(state == 0 or state == 1) {
                qs.setoLang(state);
            } else {
                sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
            }
        } else {
            defformatstring(msg)("%s", (qs.isLangWarnOn() == 1) ? "On" : "Off");
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, msg);
        }
    }
        
	VAR(ircignore, 0, 0, 1);
	SVAR(contactemail, "");
    QSERV_CALLBACK callops_cmd(p) {
    	//IRC enabled, notify irc-operators/masters/admins
    	if(!getvar("ircignore")) {
    		out(ECHO_IRC, "[Attention operator(s)]: %s: %s is in need of assistance.", ircoperators, CMD_SCI.name); 
    		defformatstring(toclient)("You alerted IRC operator(s): %s", ircoperators); 
    		sendf(CMD_SENDER, 1, "ris", N_SERVMSG, toclient);
    		loopv(clients) {
        		clientinfo *ci = clients[i];
        		defformatstring(s)("\f6[Attention]: %s, %s is in need of assistance.", colorname(ci), CMD_SCI.name); 
        		if((ci->privilege == PRIV_ADMIN || ci->privilege == PRIV_MASTER) && ci->connected && ci->clientnum != CMD_SENDER) sendf(ci->clientnum, 1, "ris", N_SERVMSG, s);
        	}
        }
        //IRC disabled, echo to console and notify masters/admins
        else if(getvar("ircignore")) {
        	defformatstring(toclient)("\f7Admins have been notified. \nEmail: \f1%s \f7for more assistance.",contactemail); 
        	sendf(CMD_SENDER, 1, "ris", N_SERVMSG, toclient);
        	out(ECHO_CONSOLE, "[Attention operator(s)]: %s: %s is in need of assistance.", ircoperators, CMD_SCI.name); 
        	loopv(clients) {
        		clientinfo *ci = clients[i];
        		defformatstring(s)("\f6[Attention]: %s, %s is in need of assistance.", colorname(ci), CMD_SCI.name); 
        		if((ci->privilege == PRIV_ADMIN || ci->privilege == PRIV_MASTER) && ci->connected && ci->clientnum != CMD_SENDER) sendf(ci->clientnum, 1, "ris", N_SERVMSG, s);
        	}
        }
        //Variable for ircignore function handler, set ircignore to -1 to disable master/admin/irc notifications
        else {
        	defformatstring(toclient)("\f7Sorry, No operators are available currently. \nEmail: \f1%s \f7for more assistance.",contactemail); 
        	sendf(CMD_SENDER, 1, "ris", N_SERVMSG, toclient);
        }
    }
    
    SVAR(qserv_version, "");
    QSERV_CALLBACK getversion_cmd(p) {
    defformatstring(ver)("\f7Running \f3QServ \f7(\f2%s\f7): \f1www.github.com/deathstar/QServCollect", qserv_version);
    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, ver);
    }
    
    QSERV_CALLBACK forceintermission_cmd(p) {bool intermission = false; if(!intermission){startintermission(); defformatstring(msg)("\f0%s \f7forced an intermission",CMD_SCI.name);sendf(-1, 1, "ris", N_SERVMSG, msg); out(ECHO_IRC,"%s forced an intermission",CMD_SCI.name);}}

    QSERV_CALLBACK me_cmd(p) {
        if(strlen(fulltext) > 0) {
            qs.checkoLang(CMD_SENDER, fulltext);
            defformatstring(msg)("\f0%s \f7%s", CMD_SCI.name, fulltext);
            sendf(-1, 1, "ris", N_SERVMSG, msg);
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
    
    QSERV_CALLBACK echo_cmd(p) {
        if(strlen(fulltext) > 0) {
            out(ECHO_SERV, "%s", fulltext);
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
    
    QSERV_CALLBACK pm_cmd(p) {
        int cn = -1;
        if(args[1] != NULL && strlen(fulltext) > 0) {
            cn = atoi(args[1]);
            clientinfo *ci = qs.getClient(cn);
            if(strlen(fulltext) > 0 && cn!=CMD_SENDER && ci != NULL && cn >= 0 && cn <= 128 && args[1] != NULL) {
                clientinfo *self = qs.getClient(CMD_SENDER);
                if(strlen(fulltext) > 0 && ci->connected && args[1] != NULL && ci != NULL) {
                    char* privatemessage = fulltext + 1; //ommit cn from fulltext
                    defformatstring(recieverpmmsg)("\f7Private message from \f0%s\f7:\f3%s", colorname(self), privatemessage);
                    sendf(cn, 1, "ris", N_SERVMSG, recieverpmmsg);
                    defformatstring(senderpmconf)("\f7Sent \f0%s \f7your message:\f3%s", colorname(ci), privatemessage);
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, senderpmconf);
                }
            }
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
    
    QSERV_CALLBACK sendprivs_cmd(p) {
        int cn = -1;
        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                clientinfo *ci = qs.getClient(cn);
                clientinfo *self = qs.getClient(CMD_SENDER);
                
                if(ci != NULL) {
                    if(ci->connected) {
                        defformatstring(shareprivsmsg)("\f7Ok, %s\f7. Sharing your privileges with \f0%s\f7.", colorname(self), colorname(ci));
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, shareprivsmsg);
                        if(self->privilege==PRIV_MASTER) {
                            defformatstring(sendprivsmsg)("\f7You have received \f0master \f7from \f0%s\f7.", colorname(self));
                            sendf(cn, 1, "ris", N_SERVMSG, sendprivsmsg);
                            server::setmaster(ci, 1, "", NULL, NULL, PRIV_MASTER, true, false, false);
                        }
                        else if(self->privilege==PRIV_ADMIN) {
                            defformatstring(sendprivsmsg)("\f7You have received \f6admin \f7from \f6%s\f7.", colorname(self));
                            sendf(cn, 1, "ris", N_SERVMSG, sendprivsmsg);
                            server::setmaster(ci, 1, "", NULL, NULL, PRIV_ADMIN, true, false, false);
                        }
                    }
                } else {
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                } 
            }
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
    
    QSERV_CALLBACK help_cmd(p) {
        if(CMD_SA) {
            int lastcmd = -1;
            char command[50];
            
            sprintf(command, "%c%s", commandprefix, args[1]);
            if(strlen(command) > 0) {
                for(int i = 0; i < CMD_LAST; i++) {
                    if(!strcmp(CMD_NAME(i), command)) {
                        lastcmd = i;
                        break;
                    }
                }
            }
            
            if(CMD_SCI.privilege >= qs.getCommandPriv(lastcmd)) {
                if(lastcmd > -1) {
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(lastcmd));
                } else {
                    sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Command not found. \nUse \f2\"#help\" \f3for a list of commands or \f2\"#help <name-ofcommand>\" \f3for usage.");
                }
            } else {
                sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Insufficient permissions to view command info");
            }
        } else {
            static char colors[2], commandList[2056] = {0};
            int color = -1;
            strcpy(commandList, "");
            clientinfo *ci = qs.getClient(CMD_SENDER);
            if(ci->privilege==PRIV_ADMIN) {
                sprintf(commandList, "%s", "\f2Commands: \f7");
                for(int i = 0; i < CMD_LAST; i++) {
                    if(CMD_PRIV(i) == PRIV_ADMIN || CMD_PRIV(i) == PRIV_MASTER || CMD_PRIV(i) == PRIV_NONE) {
                        strcat(commandList, CMD_NAME(i));
                        if(i != CMD_LAST-1) {
                            strcat(commandList, " ");
                        }
                    }
                }
            }
            else if(ci->privilege==PRIV_MASTER) {
                sprintf(commandList, "%s", "\f2Commands: \f7");
                for(int i = 0; i < CMD_LAST; i++) {
                    if(CMD_PRIV(i) == PRIV_MASTER || CMD_PRIV(i) == PRIV_NONE) {
                        strcat(commandList, CMD_NAME(i));
                        if(i != CMD_LAST-1) {
                            strcat(commandList, " ");
                        }
                    }
                }
            }
            else if(ci->privilege==PRIV_NONE) {
                sprintf(commandList, "%s", "\f2Commands: \f7");
                for(int i = 0; i < CMD_LAST; i++) {
                    if(CMD_PRIV(i) == PRIV_NONE) {
                        strcat(commandList, CMD_NAME(i));
                        if(i != CMD_LAST-1) {
                            strcat(commandList, " ");
                        }
                    }
                }
                
                /*for(int i = 0; i < CMD_LAST; i++) {
                 if(CMD_PRIV(i) == PRIV_NONE) {
                 color = 7;
                 } else if(CMD_PRIV(i) == PRIV_MASTER) {
                 color = 0;
                 } else if(CMD_PRIV(i) == PRIV_ADMIN) {
                 color = 6;
                 }
                 sprintf(colors, "\f%d", color);
                 strcat(commandList, colors);
                 //strcat(commandList, CMD_NAME(i));
                 }
                 */
            }
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, commandList);
        }
    }
    
#include <iostream>
#include <string>
#include <stdio.h>
#include <time.h>
    QSERV_CALLBACK localtime_cmd(p) {
  		time_t rawtime; time (&rawtime);
  		defformatstring(localtime)("Local server time: %s", ctime (&rawtime));
        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, localtime);
    }

    QSERV_CALLBACK time_cmd(p) {
        int UTCOffset = -1;
        if(CMD_SA) {
            UTCOffset = atoi(args[1]);
            if(UTCOffset != NULL && args[1] != NULL) {
                #define UTC (0)
                time_t rawtime;
                struct tm * ptm;
                time ( &rawtime );
                ptm = gmtime ( &rawtime );
                //tm_isdst is not accurate nor reliable for UTC applications. UTC time is tm_hour without UTCOffset variable
                defformatstring(TimeOffset)("Time for UTC (%d): %2d:%02d\n", UTCOffset, (ptm->tm_hour+UTCOffset)%24, ptm->tm_min);
                defformatstring(TimeOffsetDST)("Time for UTC (%d) with Daylight Savings Time: %2d:%02d\n", UTCOffset, (ptm->tm_hour+UTCOffset+1)%24, ptm->tm_min);
                defformatstring(TimeOutput)("%s%s", TimeOffset, TimeOffsetDST);
                sendf(CMD_SENDER, 1, "ris", N_SERVMSG, TimeOutput);
            }
            else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "Please enter a time zone number");
        }
        else sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));

    }

    QSERV_CALLBACK bunny_cmd(p) {
        if(strlen(fulltext) > 0) {
            
            defformatstring(msg)("%s \f0%s: \f7%s", bunny, "Tip", fulltext);
            sendf(-1, 1, "ris", N_SERVMSG, msg);
        } else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }

    QSERV_CALLBACK owords_cmd(p) {
        char owordList[1024] = "Offensive words (words not to say): ";
        char colors[10];

        for(int i = 0; i < 50; i++) {
            if(strlen(owords[i]) > 0) {
                sprintf(colors, "\f%d", 3);
                strcat(owordList, colors);
                strcat(owordList, owords[i]);
                strcat(owordList, "\f2, ");
            }
        }
        owordList[strlen(owordList)-2] = '\0';
        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, owordList);
    }

     QSERV_CALLBACK stats_cmd(p) {
         #include <stdio.h>
         //check to see if we want to use curl
         FILE* f_mode = fopen("usecurl.txt", "r");
         bool curlgeolocation;
         if(f_mode) {curlgeolocation = true;}
         else {curlgeolocation = false;}
         
        bool usage = false;
        int cn = -1;

        if(CMD_SA) {
            cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
                //if(!isalpha(cn)) {
                    sendstats:
                    clientinfo *ci = qs.getClient(cn);

                    if(ci != NULL) {
                        if(ci->connected) {
                            char *ip = toip(cn), lmsg[3][255];
                            const char *location;

                            if(!strcmp("127.0.0.1", ip)) {
                                location = (char*)"localhost";
                            } else {
                                location = qs.cgip(ip).c_str(); //qs.congeoip(ip); <-- just country

                                if(!location) {
                                    sprintf(lmsg[1], "%s", "Unknown Location");
                                }
                            }
                            string buf;
                            char msg[MAXTRANS];
                            
                            formatstring(msg)("\f7Stats for \f0%s\f7: cn: \f2%d \f7frags: \f2%i \f7deaths: \f2%i \f7suicides: \f2%i \f7kpd: \f2%.2f \f7acc: \f2%i%%",
                                              colorname(ci), ci->clientnum, ci->state.frags, ci->state.deaths, ci->state._suicides,
                                              (float(ci->state.frags)/float(max(ci->state.deaths, 1))), ci->state.damage*100/max(ci->state.shotdamage,1));
                            if(server::q_teammode)
                            {
                                formatstring(buf)("\n\f7teamkills: \f2%i \f7flags scored: \f2%i \f7flags stolen: \f2%i \f7flags returned: \f2%i", ci->state.teamkills, ci->state.flags, ci->state._stolen, ci->state._returned);
                                concatstring(msg, buf, MAXTRANS);
                            }
                            formatstring(buf)("\n\f7shotgun: \f2%i%% \f7chaingun: \f2%i%% \f7rocketlauncher: \f2%i%% \f7rifle: \f2%i%% \f7grenadelauncher: \f2%i%% \f7pistol: \f2%i%%",
                            getwepaccuracy(ci->clientnum, 1), getwepaccuracy(ci->clientnum, 2), getwepaccuracy(ci->clientnum, 3),
                            getwepaccuracy(ci->clientnum, 4), getwepaccuracy(ci->clientnum, 5), getwepaccuracy(ci->clientnum, 6));
                            concatstring(msg, buf, MAXTRANS);
                            
                            if(location) sprintf(lmsg[1], "%s", location);
                            (CMD_SCI.privilege == PRIV_ADMIN) ? sprintf(lmsg[0], "%s (%s)", lmsg[1], ip) :
                            sprintf(lmsg[0], "%s", lmsg[1]);
                            
                            if(curlgeolocation) {
                            }
                            else {
                                formatstring(buf)("\n\f7location: \f6%s", lmsg[0]);
                                concatstring(msg, buf, MAXTRANS);
                            }
                            
                            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, msg);
                            
                            send_connected_time(ci, CMD_SENDER); 
                        }
                    } else {
                        sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                    }
                /*} else {
                    usage = true;
                }*/
            } else {
                usage = true;
            }
        } else {
            cn = CMD_SENDER;
            goto sendstats;
        }

        if(usage) sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
    }
	
                    
	QSERV_CALLBACK revokepriv_cmd(p) {
		int cn = -1;
	
        if(CMD_SA) {
			cn = atoi(args[1]);
            if(cn >= 0 && cn <= 1000) {
				clientinfo *ci = qs.getClient(cn);
				clientinfo *cisender = qs.getClient(CMD_SENDER);
				
                if(ci != NULL) {
					if(ci->connected) {
						server::revokemaster(ci);
						defformatstring(msg)("Privileges have been revoked from the specified client \f3%s", colorname(ci));
						sendf(-1, 1, "ris", N_SERVMSG, msg);
						setmaster(ci, true, "", NULL, NULL, PRIV_NONE, true, false, true);
					}
				} else {
					sendf(CMD_SENDER, 1, "ris", N_SERVMSG, "\f3Error: Player not connected");
                } 
			}
		} else {
            sendf(CMD_SENDER, 1, "ris", N_SERVMSG, CMD_DESC(cid));
        }
    }
}



