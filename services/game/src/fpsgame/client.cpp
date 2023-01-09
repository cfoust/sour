#include "game.h"

#if __EMSCRIPTEN__
#include <emscripten.h>
#endif

#if __EMSCRIPTEN__
    extern bool sendsouritems = false; 
#endif

const char * gettype(int type) {
	switch (type) {
	case N_CONNECT:
		return "N_CONNECT";
	case N_SERVINFO:
		return "N_SERVINFO";
	case N_WELCOME:
		return "N_WELCOME";
	case N_INITCLIENT:
		return "N_INITCLIENT";
	case N_POS:
		return "N_POS";
	case N_TEXT:
		return "N_TEXT";
	case N_SOUND:
		return "N_SOUND";
	case N_CDIS:
		return "N_CDIS";
	case N_SHOOT:
		return "N_SHOOT";
	case N_EXPLODE:
		return "N_EXPLODE";
	case N_SUICIDE:
		return "N_SUICIDE";
	case N_DIED:
		return "N_DIED";
	case N_DAMAGE:
		return "N_DAMAGE";
	case N_HITPUSH:
		return "N_HITPUSH";
	case N_SHOTFX:
		return "N_SHOTFX";
	case N_EXPLODEFX:
		return "N_EXPLODEFX";
	case N_TRYSPAWN:
		return "N_TRYSPAWN";
	case N_SPAWNSTATE:
		return "N_SPAWNSTATE";
	case N_SPAWN:
		return "N_SPAWN";
	case N_FORCEDEATH:
		return "N_FORCEDEATH";
	case N_GUNSELECT:
		return "N_GUNSELECT";
	case N_TAUNT:
		return "N_TAUNT";
	case N_MAPCHANGE:
		return "N_MAPCHANGE";
	case N_MAPVOTE:
		return "N_MAPVOTE";
	case N_TEAMINFO:
		return "N_TEAMINFO";
	case N_ITEMSPAWN:
		return "N_ITEMSPAWN";
	case N_ITEMPICKUP:
		return "N_ITEMPICKUP";
	case N_ITEMACC:
		return "N_ITEMACC";
	case N_TELEPORT:
		return "N_TELEPORT";
	case N_JUMPPAD:
		return "N_JUMPPAD";
	case N_PING:
		return "N_PING";
	case N_PONG:
		return "N_PONG";
	case N_CLIENTPING:
		return "N_CLIENTPING";
	case N_TIMEUP:
		return "N_TIMEUP";
	case N_FORCEINTERMISSION:
		return "N_FORCEINTERMISSION";
	case N_SERVMSG:
		return "N_SERVMSG";
	case N_ITEMLIST:
		return "N_ITEMLIST";
	case N_RESUME:
		return "N_RESUME";
	case N_EDITMODE:
		return "N_EDITMODE";
	case N_EDITENT:
		return "N_EDITENT";
	case N_EDITF:
		return "N_EDITF";
	case N_EDITT:
		return "N_EDITT";
	case N_EDITM:
		return "N_EDITM";
	case N_FLIP:
		return "N_FLIP";
	case N_COPY:
		return "N_COPY";
	case N_PASTE:
		return "N_PASTE";
	case N_ROTATE:
		return "N_ROTATE";
	case N_REPLACE:
		return "N_REPLACE";
	case N_DELCUBE:
		return "N_DELCUBE";
	case N_REMIP:
		return "N_REMIP";
	case N_EDITVSLOT:
		return "N_EDITVSLOT";
	case N_UNDO:
		return "N_UNDO";
	case N_REDO:
		return "N_REDO";
	case N_NEWMAP:
		return "N_NEWMAP";
	case N_GETMAP:
		return "N_GETMAP";
	case N_SENDMAP:
		return "N_SENDMAP";
	case N_CLIPBOARD:
		return "N_CLIPBOARD";
	case N_EDITVAR:
		return "N_EDITVAR";
	case N_MASTERMODE:
		return "N_MASTERMODE";
	case N_KICK:
		return "N_KICK";
	case N_CLEARBANS:
		return "N_CLEARBANS";
	case N_CURRENTMASTER:
		return "N_CURRENTMASTER";
	case N_SPECTATOR:
		return "N_SPECTATOR";
	case N_SETMASTER:
		return "N_SETMASTER";
	case N_SETTEAM:
		return "N_SETTEAM";
	case N_BASES:
		return "N_BASES";
	case N_BASEINFO:
		return "N_BASEINFO";
	case N_BASESCORE:
		return "N_BASESCORE";
	case N_REPAMMO:
		return "N_REPAMMO";
	case N_BASEREGEN:
		return "N_BASEREGEN";
	case N_ANNOUNCE:
		return "N_ANNOUNCE";
	case N_LISTDEMOS:
		return "N_LISTDEMOS";
	case N_SENDDEMOLIST:
		return "N_SENDDEMOLIST";
	case N_GETDEMO:
		return "N_GETDEMO";
	case N_SENDDEMO:
		return "N_SENDDEMO";
	case N_DEMOPLAYBACK:
		return "N_DEMOPLAYBACK";
	case N_RECORDDEMO:
		return "N_RECORDDEMO";
	case N_STOPDEMO:
		return "N_STOPDEMO";
	case N_CLEARDEMOS:
		return "N_CLEARDEMOS";
	case N_TAKEFLAG:
		return "N_TAKEFLAG";
	case N_RETURNFLAG:
		return "N_RETURNFLAG";
	case N_RESETFLAG:
		return "N_RESETFLAG";
	case N_INVISFLAG:
		return "N_INVISFLAG";
	case N_TRYDROPFLAG:
		return "N_TRYDROPFLAG";
	case N_DROPFLAG:
		return "N_DROPFLAG";
	case N_SCOREFLAG:
		return "N_SCOREFLAG";
	case N_INITFLAGS:
		return "N_INITFLAGS";
	case N_SAYTEAM:
		return "N_SAYTEAM";
	case N_CLIENT:
		return "N_CLIENT";
	case N_AUTHTRY:
		return "N_AUTHTRY";
	case N_AUTHKICK:
		return "N_AUTHKICK";
	case N_AUTHCHAL:
		return "N_AUTHCHAL";
	case N_AUTHANS:
		return "N_AUTHANS";
	case N_REQAUTH:
		return "N_REQAUTH";
	case N_PAUSEGAME:
		return "N_PAUSEGAME";
	case N_GAMESPEED:
		return "N_GAMESPEED";
	case N_ADDBOT:
		return "N_ADDBOT";
	case N_DELBOT:
		return "N_DELBOT";
	case N_INITAI:
		return "N_INITAI";
	case N_FROMAI:
		return "N_FROMAI";
	case N_BOTLIMIT:
		return "N_BOTLIMIT";
	case N_BOTBALANCE:
		return "N_BOTBALANCE";
	case N_MAPCRC:
		return "N_MAPCRC";
	case N_CHECKMAPS:
		return "N_CHECKMAPS";
	case N_SWITCHNAME:
		return "N_SWITCHNAME";
	case N_SWITCHMODEL:
		return "N_SWITCHMODEL";
	case N_SWITCHTEAM:
		return "N_SWITCHTEAM";
	case N_INITTOKENS:
		return "N_INITTOKENS";
	case N_TAKETOKEN:
		return "N_TAKETOKEN";
	case N_EXPIRETOKENS:
		return "N_EXPIRETOKENS";
	case N_DROPTOKENS:
		return "N_DROPTOKENS";
	case N_DEPOSITTOKENS:
		return "N_DEPOSITTOKENS";
	case N_STEALTOKENS:
		return "N_STEALTOKENS";
	case N_SERVCMD:
		return "N_SERVCMD";
	case N_DEMOPACKET:
		return "N_DEMOPACKET";
	case NUMMSG:
		return "NUMMSG";
	default:
		return "";
	}
}


namespace game
{
    VARP(minradarscale, 0, 384, 10000);
    VARP(maxradarscale, 1, 1024, 10000);
    VARP(radarteammates, 0, 1, 1);
    FVARP(minimapalpha, 0, 1, 1);

    float calcradarscale()
    {
        return clamp(max(minimapradius.x, minimapradius.y)/3, float(minradarscale), float(maxradarscale));
    }

    void drawminimap(fpsent *d, float x, float y, float s)
    {
        vec pos = vec(d->o).sub(minimapcenter).mul(minimapscale).add(0.5f), dir;
        vecfromyawpitch(camera1->yaw, 0, 1, 0, dir);
        float scale = calcradarscale();
        gle::defvertex(2);
        gle::deftexcoord0();
        gle::begin(GL_TRIANGLE_FAN);
        loopi(16)
        {
            vec v = vec(0, -1, 0).rotate_around_z(i/16.0f*2*M_PI);
            gle::attribf(x + 0.5f*s*(1.0f + v.x), y + 0.5f*s*(1.0f + v.y));
            vec tc = vec(dir).rotate_around_z(i/16.0f*2*M_PI);
            gle::attribf(pos.x + tc.x*scale*minimapscale.x, pos.y + tc.y*scale*minimapscale.y);
        }
        gle::end();
    }

    void drawradar(float x, float y, float s)
    {
        gle::defvertex(2);
        gle::deftexcoord0();
        gle::begin(GL_TRIANGLE_STRIP);
        gle::attribf(x,   y);   gle::attribf(0, 0);
        gle::attribf(x+s, y);   gle::attribf(1, 0);
        gle::attribf(x,   y+s); gle::attribf(0, 1);
        gle::attribf(x+s, y+s); gle::attribf(1, 1);
        gle::end();
    }

    void drawteammate(fpsent *d, float x, float y, float s, fpsent *o, float scale)
    {
        vec dir = d->o;
        dir.sub(o->o).div(scale);
        float dist = dir.magnitude2(), maxdist = 1 - 0.05f - 0.05f;
        if(dist >= maxdist) dir.mul(maxdist/dist);
        dir.rotate_around_z(-camera1->yaw*RAD);
        float bs = 0.06f*s,
              bx = x + s*0.5f*(1.0f + dir.x),
              by = y + s*0.5f*(1.0f + dir.y);
        vec v(-0.5f, -0.5f, 0);
        v.rotate_around_z((90+o->yaw-camera1->yaw)*RAD);
        gle::attribf(bx + bs*v.x, by + bs*v.y); gle::attribf(0, 0);
        gle::attribf(bx + bs*v.y, by - bs*v.x); gle::attribf(1, 0);
        gle::attribf(bx - bs*v.x, by - bs*v.y); gle::attribf(1, 1);
        gle::attribf(bx - bs*v.y, by + bs*v.x); gle::attribf(0, 1);
    }

    void drawteammates(fpsent *d, float x, float y, float s)
    {
        if(!radarteammates) return;
        float scale = calcradarscale();
        int alive = 0, dead = 0;
        loopv(players) 
        {
            fpsent *o = players[i];
            if(o != d && o->state == CS_ALIVE && isteam(o->team, d->team))
            {
                if(!alive++) 
                {
                    settexture(isteam(d->team, player1->team) ? "packages/hud/blip_blue_alive.png" : "packages/hud/blip_red_alive.png");
                    gle::defvertex(2);
                    gle::deftexcoord0();
                    gle::begin(GL_QUADS);
                }
                drawteammate(d, x, y, s, o, scale);
            }
        }
        if(alive) gle::end();
        loopv(players) 
        {
            fpsent *o = players[i];
            if(o != d && o->state == CS_DEAD && isteam(o->team, d->team))
            {
                if(!dead++) 
                {
                    settexture(isteam(d->team, player1->team) ? "packages/hud/blip_blue_dead.png" : "packages/hud/blip_red_dead.png");
                    gle::defvertex(2);
                    gle::deftexcoord0();
                    gle::begin(GL_QUADS);
                }
                drawteammate(d, x, y, s, o, scale);
            }
        }
        if(dead) gle::end();
    }
        
    #include "capture.h"
    #include "ctf.h"
    #include "collect.h"

    clientmode *cmode = NULL;
    captureclientmode capturemode;
    ctfclientmode ctfmode;
    collectclientmode collectmode;

    void setclientmode()
    {
        if(m_capture) cmode = &capturemode;
        else if(m_ctf) cmode = &ctfmode;
        else if(m_collect) cmode = &collectmode;
        else cmode = NULL;
    }

    bool senditemstoserver = false, sendcrc = false; // after a map change, since server doesn't have map data
    int lastping = 0;

    bool connected = false, remote = false, demoplayback = false, gamepaused = false;
    int sessionid = 0, mastermode = MM_OPEN, gamespeed = 100;
    string servinfo = "", servauth = "", connectpass = "";

    VARP(deadpush, 1, 2, 20);

    void switchname(const char *name)
    {
        filtertext(player1->name, name, false, false, MAXNAMELEN);
        if(!player1->name[0]) {
			copystring(player1->name, "unnamed");
			conoutf("setting name to: %s", player1->name);
		}
        addmsg(N_SWITCHNAME, "rs", player1->name);
    }
    void printname()
    {
        conoutf("your name is: %s", colorname(player1));
    }
    ICOMMAND(name, "sN", (char *s, int *numargs),
    {
        if(*numargs > 0) switchname(s);
        else if(!*numargs) printname();
        else result(colorname(player1));
    });
    ICOMMAND(getname, "", (), result(player1->name));

    void switchteam(const char *team)
    {
        if(player1->clientnum < 0) filtertext(player1->team, team, false, false, MAXTEAMLEN);
        else addmsg(N_SWITCHTEAM, "rs", team);
    }
    void printteam()
    {
        conoutf("your team is: %s", player1->team);
    }
    ICOMMAND(team, "sN", (char *s, int *numargs),
    {
        if(*numargs > 0) switchteam(s);
        else if(!*numargs) printteam();
        else result(player1->team);
    });
    ICOMMAND(getteam, "", (), result(player1->team));

    void switchplayermodel(int playermodel)
    {
        player1->playermodel = playermodel;
        addmsg(N_SWITCHMODEL, "ri", player1->playermodel);
    }

    struct authkey
    {
        char *name, *key, *desc;
        int lastauth;

        authkey(const char *name, const char *key, const char *desc)
            : name(newstring(name)), key(newstring(key)), desc(newstring(desc)),
              lastauth(0)
        {
        }

        ~authkey()
        {
            DELETEA(name);
            DELETEA(key);
            DELETEA(desc);
        }
    };
    vector<authkey *> authkeys;

    authkey *findauthkey(const char *desc = "")
    {
        loopv(authkeys) if(!strcmp(authkeys[i]->desc, desc) && !strcasecmp(authkeys[i]->name, player1->name)) return authkeys[i];
        loopv(authkeys) if(!strcmp(authkeys[i]->desc, desc)) return authkeys[i];
        return NULL;
    }

    VARP(autoauth, 0, 1, 1);

    void addauthkey(const char *name, const char *key, const char *desc)
    {
        loopvrev(authkeys) if(!strcmp(authkeys[i]->desc, desc) && !strcmp(authkeys[i]->name, name)) delete authkeys.remove(i);
        if(name[0] && key[0]) authkeys.add(new authkey(name, key, desc));
    }
    ICOMMAND(authkey, "sss", (char *name, char *key, char *desc), addauthkey(name, key, desc));

    bool hasauthkey(const char *name, const char *desc)
    {
        if(!name[0] && !desc[0]) return authkeys.length() > 0;
        loopvrev(authkeys) if(!strcmp(authkeys[i]->desc, desc) && !strcmp(authkeys[i]->name, name)) return true;
        return false;
    }

    ICOMMAND(hasauthkey, "ss", (char *name, char *desc), intret(hasauthkey(name, desc) ? 1 : 0));

    void genauthkey(const char *secret)
    {
        if(!secret[0]) { conoutf(CON_ERROR, "you must specify a secret password"); return; }
        vector<char> privkey, pubkey;
        genprivkey(secret, privkey, pubkey);
        conoutf("private key: %s", privkey.getbuf());
        conoutf("public key: %s", pubkey.getbuf());
        result(privkey.getbuf());
    }
    COMMAND(genauthkey, "s");

    void getpubkey(const char *desc)
    {
        authkey *k = findauthkey(desc);
        if(!k) { if(desc[0]) conoutf(CON_ERROR, "no authkey found: %s", desc); else conoutf(CON_ERROR, "no global authkey found"); return; }
        vector<char> pubkey;
        if(!calcpubkey(k->key, pubkey)) { conoutf(CON_ERROR, "failed calculating pubkey"); return; }
        result(pubkey.getbuf());
    }
    COMMAND(getpubkey, "s");

    void saveauthkeys()
    {
        stream *f = openfile("auth.cfg", "w");
        if(!f) { conoutf(CON_ERROR, "failed to open auth.cfg for writing"); return; }
        loopv(authkeys)
        {
            authkey *a = authkeys[i];
            f->printf("authkey %s %s %s\n", escapestring(a->name), escapestring(a->key), escapestring(a->desc));
        }
        conoutf("saved authkeys to auth.cfg");
        delete f;
    }
    COMMAND(saveauthkeys, "");

    void sendmapinfo()
    {
        if(!connected) return;
        sendcrc = true;
        if(player1->state!=CS_SPECTATOR || player1->privilege || !remote) senditemstoserver = true;
    }

    void writeclientinfo(stream *f)
    {
        f->printf("name %s\n", escapestring(player1->name));
    }

    bool allowedittoggle()
    {
        if(editmode) return true;
        if(isconnected() && multiplayer(false) && !m_edit)
        {
            conoutf(CON_ERROR, "editing in multiplayer requires coop edit mode (1)");
            return false;
        }
        return execidentbool("allowedittoggle", true);
    }

    void edittoggled(bool on)
    {
        addmsg(N_EDITMODE, "ri", on ? 1 : 0);
        if(player1->state==CS_DEAD) deathstate(player1, true);
        else if(player1->state==CS_EDITING && player1->editstate==CS_DEAD) showscores(false);
        disablezoom();
        player1->suicided = player1->respawned = -2;
    }

    const char *getclientname(int cn)
    {
        fpsent *d = getclient(cn);
        return d ? d->name : "";
    }
    ICOMMAND(getclientname, "i", (int *cn), result(getclientname(*cn)));

    const char *getclientteam(int cn)
    {
        fpsent *d = getclient(cn);
        return d ? d->team : "";
    }
    ICOMMAND(getclientteam, "i", (int *cn), result(getclientteam(*cn)));

    int getclientmodel(int cn)
    {
        fpsent *d = getclient(cn);
        return d ? d->playermodel : -1;
    }
    ICOMMAND(getclientmodel, "i", (int *cn), intret(getclientmodel(*cn)));

    const char *getclienticon(int cn)
    {
        fpsent *d = getclient(cn);
        if(!d || d->state==CS_SPECTATOR) return "spectator";
        const playermodelinfo &mdl = getplayermodelinfo(d);
        return m_teammode ? (isteam(player1->team, d->team) ? mdl.blueicon : mdl.redicon) : mdl.ffaicon;
    }
    ICOMMAND(getclienticon, "i", (int *cn), result(getclienticon(*cn)));

    bool ismaster(int cn)
    {
        fpsent *d = getclient(cn);
        return d && d->privilege >= PRIV_MASTER;
    }
    ICOMMAND(ismaster, "i", (int *cn), intret(ismaster(*cn) ? 1 : 0));

    bool isauth(int cn)
    {
        fpsent *d = getclient(cn);
        return d && d->privilege >= PRIV_AUTH;
    }
    ICOMMAND(isauth, "i", (int *cn), intret(isauth(*cn) ? 1 : 0));

    bool isadmin(int cn)
    {
        fpsent *d = getclient(cn);
        return d && d->privilege >= PRIV_ADMIN;
    }
    ICOMMAND(isadmin, "i", (int *cn), intret(isadmin(*cn) ? 1 : 0));

    ICOMMAND(getmastermode, "", (), intret(mastermode));
    ICOMMAND(mastermodename, "i", (int *mm), result(server::mastermodename(*mm, "")));

    bool isspectator(int cn)
    {
        fpsent *d = getclient(cn);
        return d && d->state==CS_SPECTATOR;
    }
    ICOMMAND(isspectator, "i", (int *cn), intret(isspectator(*cn) ? 1 : 0));

    bool isai(int cn, int type)
    {
        fpsent *d = getclient(cn);
        int aitype = type > 0 && type < AI_MAX ? type : AI_BOT;
        return d && d->aitype==aitype;
    }
    ICOMMAND(isai, "ii", (int *cn, int *type), intret(isai(*cn, *type) ? 1 : 0));

    int parseplayer(const char *arg)
    {
        char *end;
        int n = strtol(arg, &end, 10);
        if(*arg && !*end)
        {
            if(n!=player1->clientnum && !clients.inrange(n)) return -1;
            return n;
        }
        // try case sensitive first
        loopv(players)
        {
            fpsent *o = players[i];
            if(!strcmp(arg, o->name)) return o->clientnum;
        }
        // nothing found, try case insensitive
        loopv(players)
        {
            fpsent *o = players[i];
            if(!strcasecmp(arg, o->name)) return o->clientnum;
        }
        return -1;
    }
    ICOMMAND(getclientnum, "s", (char *name), intret(name[0] ? parseplayer(name) : player1->clientnum));

    void listclients(bool local, bool bots)
    {
        vector<char> buf;
        string cn;
        int numclients = 0;
        if(local && connected)
        {
            formatstring(cn, "%d", player1->clientnum);
            buf.put(cn, strlen(cn));
            numclients++;
        }
        loopv(clients) if(clients[i] && (bots || clients[i]->aitype == AI_NONE))
        {
            formatstring(cn, "%d", clients[i]->clientnum);
            if(numclients++) buf.add(' ');
            buf.put(cn, strlen(cn));
        }
        buf.add('\0');
        result(buf.getbuf());
    }
    ICOMMAND(listclients, "bb", (int *local, int *bots), listclients(*local>0, *bots!=0));

    void clearbans()
    {
        addmsg(N_CLEARBANS, "r");
    }
    COMMAND(clearbans, "");

    void kick(const char *victim, const char *reason)
    {
        int vn = parseplayer(victim);
        if(vn>=0 && vn!=player1->clientnum) addmsg(N_KICK, "ris", vn, reason);
    }
    COMMAND(kick, "ss");

    void authkick(const char *desc, const char *victim, const char *reason)
    {
        authkey *a = findauthkey(desc);
        int vn = parseplayer(victim);
        if(a && vn>=0 && vn!=player1->clientnum) 
        {
            a->lastauth = lastmillis;
            addmsg(N_AUTHKICK, "rssis", a->desc, a->name, vn, reason);
        }
    }
    ICOMMAND(authkick, "ss", (const char *victim, const char *reason), authkick("", victim, reason));
    ICOMMAND(sauthkick, "ss", (const char *victim, const char *reason), if(servauth[0]) authkick(servauth, victim, reason));
    ICOMMAND(dauthkick, "sss", (const char *desc, const char *victim, const char *reason), if(desc[0]) authkick(desc, victim, reason));

    vector<int> ignores;

    void ignore(int cn)
    {
        fpsent *d = getclient(cn);
        if(!d || d == player1) return;
        conoutf("ignoring %s", d->name);
        if(ignores.find(cn) < 0) ignores.add(cn);
    }

    void unignore(int cn)
    {
        if(ignores.find(cn) < 0) return;
        fpsent *d = getclient(cn);
        if(d) conoutf("stopped ignoring %s", d->name);
        ignores.removeobj(cn);
    }

    bool isignored(int cn) { return ignores.find(cn) >= 0; }

    ICOMMAND(ignore, "s", (char *arg), ignore(parseplayer(arg)));
    ICOMMAND(unignore, "s", (char *arg), unignore(parseplayer(arg))); 
    ICOMMAND(isignored, "s", (char *arg), intret(isignored(parseplayer(arg)) ? 1 : 0));

    void setteam(const char *arg1, const char *arg2)
    {
        int i = parseplayer(arg1);
        if(i>=0) addmsg(N_SETTEAM, "ris", i, arg2);
    }
    COMMAND(setteam, "ss");

    void hashpwd(const char *pwd)
    {
        if(player1->clientnum<0) return;
        string hash;
        server::hashpassword(player1->clientnum, sessionid, pwd, hash);
        result(hash);
    }
    COMMAND(hashpwd, "s");

    void setmaster(const char *arg, const char *who)
    {
        if(!arg[0]) return;
        int val = 1, cn = player1->clientnum;
        if(who[0])
        {
            cn = parseplayer(who);
            if(cn < 0) return;
        }
        string hash = "";
        if(!arg[1] && isdigit(arg[0])) val = parseint(arg);
        else 
        {
            if(cn != player1->clientnum) return;
            server::hashpassword(player1->clientnum, sessionid, arg, hash);
        }
        addmsg(N_SETMASTER, "riis", cn, val, hash);
    }
    COMMAND(setmaster, "ss");
    ICOMMAND(mastermode, "i", (int *val), addmsg(N_MASTERMODE, "ri", *val));

    bool tryauth(const char *desc)
    {
        authkey *a = findauthkey(desc);
        if(!a) return false;
        a->lastauth = lastmillis;
        addmsg(N_AUTHTRY, "rss", a->desc, a->name);
        return true;
    }
    ICOMMAND(auth, "s", (char *desc), tryauth(desc));
    ICOMMAND(sauth, "", (), if(servauth[0]) tryauth(servauth));
    ICOMMAND(dauth, "s", (char *desc), if(desc[0]) tryauth(desc));

    ICOMMAND(getservauth, "", (), result(servauth));

    void togglespectator(int val, const char *who)
    {
        int i = who[0] ? parseplayer(who) : player1->clientnum;
        if(i>=0) addmsg(N_SPECTATOR, "rii", i, val);
    }
    ICOMMAND(spectator, "is", (int *val, char *who), togglespectator(*val, who));

    ICOMMAND(checkmaps, "", (), addmsg(N_CHECKMAPS, "r"));

    int gamemode = INT_MAX, nextmode = INT_MAX;
    string clientmap = "";

    void changemapserv(const char *name, int mode)        // forced map change from the server
    {
        if(multiplayer(false) && !m_mp(mode))
        {
            conoutf(CON_ERROR, "mode %s (%d) not supported in multiplayer", server::modename(gamemode), gamemode);
            loopi(NUMGAMEMODES) if(m_mp(STARTGAMEMODE + i)) { mode = STARTGAMEMODE + i; break; }
        }

        gamemode = mode;
        nextmode = mode;
        if(editmode) toggleedit();
        if(m_demo) { entities::resetspawns(); return; }
        if((m_edit && !name[0]) || !load_world(name))
        {
            emptymap(0, true, name);
            senditemstoserver = false;
        }
        startgame(); // XXX EMSCRIPTEN: load_world is async, so this is not technically correct. but doing it from the end of load_world makes it
                     //                 get out of sync with lifesequence (server sends us the lifesequence, then later startgame makes it -1
    }

    void setmode(int mode)
    {
        if(multiplayer(false) && !m_mp(mode))
        {
            conoutf(CON_ERROR, "mode %s (%d) not supported in multiplayer",  server::modename(mode), mode);
            intret(0);
            return;
        }
        nextmode = mode;
        intret(1);
    }
    ICOMMAND(mode, "i", (int *val), setmode(*val));
    ICOMMAND(getmode, "", (), intret(gamemode));
    ICOMMAND(timeremaining, "i", (int *formatted), 
    {
        int val = max(maplimit - lastmillis + 999, 0)/1000;
        if(*formatted)
        {
            defformatstring(str, "%d:%02d", val/60, val%60);
            result(str);
        }
        else intret(val);
    });
    ICOMMANDS("m_noitems", "i", (int *mode), { int gamemode = *mode; intret(m_noitems); });
    ICOMMANDS("m_noammo", "i", (int *mode), { int gamemode = *mode; intret(m_noammo); });
    ICOMMANDS("m_insta", "i", (int *mode), { int gamemode = *mode; intret(m_insta); });
    ICOMMANDS("m_tactics", "i", (int *mode), { int gamemode = *mode; intret(m_tactics); });
    ICOMMANDS("m_efficiency", "i", (int *mode), { int gamemode = *mode; intret(m_efficiency); });
    ICOMMANDS("m_capture", "i", (int *mode), { int gamemode = *mode; intret(m_capture); });
    ICOMMANDS("m_regencapture", "i", (int *mode), { int gamemode = *mode; intret(m_regencapture); });
    ICOMMANDS("m_ctf", "i", (int *mode), { int gamemode = *mode; intret(m_ctf); });
    ICOMMANDS("m_protect", "i", (int *mode), { int gamemode = *mode; intret(m_protect); });
    ICOMMANDS("m_hold", "i", (int *mode), { int gamemode = *mode; intret(m_hold); });
    ICOMMANDS("m_collect", "i", (int *mode), { int gamemode = *mode; intret(m_collect); });
    ICOMMANDS("m_teammode", "i", (int *mode), { int gamemode = *mode; intret(m_teammode); });
    ICOMMANDS("m_demo", "i", (int *mode), { int gamemode = *mode; intret(m_demo); });
    ICOMMANDS("m_edit", "i", (int *mode), { int gamemode = *mode; intret(m_edit); });
    ICOMMANDS("m_lobby", "i", (int *mode), { int gamemode = *mode; intret(m_lobby); });
    ICOMMANDS("m_sp", "i", (int *mode), { int gamemode = *mode; intret(m_sp); });
    ICOMMANDS("m_dmsp", "i", (int *mode), { int gamemode = *mode; intret(m_dmsp); });
    ICOMMANDS("m_classicsp", "i", (int *mode), { int gamemode = *mode; intret(m_classicsp); });

    void changemap(const char *name, int mode) // request map change, server may ignore
    {
        static string s;
        if (strcmp(name, "random") == 0) {
            EM_ASM({
                Module.assets.loadRandomMap()
            });
            return;
        }

        if(!remote)
        {
            server::forcemap(name, mode);
            if(!isconnected()) localconnect();
        }
        else if(player1->state!=CS_SPECTATOR || player1->privilege) addmsg(N_MAPVOTE, "rsi", name, mode);
    }
    void changemap(const char *name)
    {
        changemap(name, m_valid(nextmode) ? nextmode : (remote ? 0 : 1));
    }
    ICOMMAND(map, "s", (char *name), changemap(name));

    void forceintermission()
    {
        if(!remote && !hasnonlocalclients()) server::startintermission();
        else addmsg(N_FORCEINTERMISSION, "r");
    }

    void forceedit(const char *name)
    {
        changemap(name, 1);
    }

    void newmap(int size)
    {
        addmsg(N_NEWMAP, "ri", size);
    }

    int needclipboard = -1;

    void sendclipboard()
    {
        uchar *outbuf = NULL;
        int inlen = 0, outlen = 0;
        if(!packeditinfo(localedit, inlen, outbuf, outlen))
        {
            outbuf = NULL;
            inlen = outlen = 0;
        }
        packetbuf p(16 + outlen, ENET_PACKET_FLAG_RELIABLE);
        putint(p, N_CLIPBOARD);
        putint(p, inlen);
        putint(p, outlen);
        if(outlen > 0) p.put(outbuf, outlen);
        sendclientpacket(p.finalize(), 1);
        needclipboard = -1;
    }

    void edittrigger(const selinfo &sel, int op, int arg1, int arg2, int arg3, const VSlot *vs)
    {
        if(m_edit) switch(op)
        {
            case EDIT_FLIP:
            case EDIT_COPY:
            case EDIT_PASTE:
            case EDIT_DELCUBE:
            {
                switch(op)
                {
                    case EDIT_COPY: needclipboard = 0; break;
                    case EDIT_PASTE:
                        if(needclipboard > 0)
                        {
                            c2sinfo(true);
                            sendclipboard();
                        }
                        break;
                }
                addmsg(N_EDITF + op, "ri9i4",
                   sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                   sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner);
                break;
            }
            case EDIT_ROTATE:
            {
                addmsg(N_EDITF + op, "ri9i5",
                   sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                   sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner,
                   arg1);
                break;
            }
            case EDIT_MAT:
            case EDIT_FACE:
            {
                addmsg(N_EDITF + op, "ri9i6",
                   sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                   sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner,
                   arg1, arg2);
                break;
            }
            case EDIT_TEX:
            {
                int tex1 = shouldpacktex(arg1);
                if(addmsg(N_EDITF + op, "ri9i6",
                    sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                    sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner,
                    tex1 ? tex1 : arg1, arg2))
                {
                    messages.pad(2);
                    int offset = messages.length();
                    if(tex1) packvslot(messages, arg1);
                    *(ushort *)&messages[offset-2] = lilswap(ushort(messages.length() - offset));
                }
                break;
            }
            case EDIT_REPLACE:
            {
                int tex1 = shouldpacktex(arg1), tex2 = shouldpacktex(arg2);
                if(addmsg(N_EDITF + op, "ri9i7",
                    sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                    sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner,
                    tex1 ? tex1 : arg1, tex2 ? tex2 : arg2, arg3))
                {
                    messages.pad(2);
                    int offset = messages.length();
                    if(tex1) packvslot(messages, arg1);
                    if(tex2) packvslot(messages, arg2);
                    *(ushort *)&messages[offset-2] = lilswap(ushort(messages.length() - offset));
                }
                break;
            }
            case EDIT_REMIP:
            {
                addmsg(N_EDITF + op, "r");
                break;
            }
            case EDIT_VSLOT:
            {
                if(addmsg(N_EDITF + op, "ri9i6",
                    sel.o.x, sel.o.y, sel.o.z, sel.s.x, sel.s.y, sel.s.z, sel.grid, sel.orient,
                    sel.cx, sel.cxs, sel.cy, sel.cys, sel.corner,
                    arg1, arg2))
                {
                    messages.pad(2);
                    int offset = messages.length();
                    packvslot(messages, vs);
                    *(ushort *)&messages[offset-2] = lilswap(ushort(messages.length() - offset));
                }
                break;
            }
            case EDIT_UNDO:
            case EDIT_REDO:
            {
                uchar *outbuf = NULL;
                int inlen = 0, outlen = 0;
                if(packundo(op, inlen, outbuf, outlen))
                {
                    if(addmsg(N_EDITF + op, "ri2", inlen, outlen)) messages.put(outbuf, outlen);
                    delete[] outbuf;
                }
                break;
            }
        }
    }

    void printvar(fpsent *d, ident *id)
    {
        if(id) switch(id->type)
        {
            case ID_VAR:
            {
                int val = *id->storage.i;
                string str;
                if(val < 0)
                    formatstring(str, "%d", val); 
                else if(id->flags&IDF_HEX && id->maxval==0xFFFFFF)
                    formatstring(str, "0x%.6X (%d, %d, %d)", val, (val>>16)&0xFF, (val>>8)&0xFF, val&0xFF);
                else
                    formatstring(str, id->flags&IDF_HEX ? "0x%X" : "%d", val);
                conoutf(CON_INFO, id->index, "%s set map var \"%s\" to %s", colorname(d), id->name, str);
                break;
            }
            case ID_FVAR:
                conoutf(CON_INFO, id->index, "%s set map var \"%s\" to %s", colorname(d), id->name, floatstr(*id->storage.f));
                break;
            case ID_SVAR:
                conoutf(CON_INFO, id->index, "%s set map var \"%s\" to \"%s\"", colorname(d), id->name, *id->storage.s);
                break;
        }
    }

    void vartrigger(ident *id)
    {
        if(!m_edit) return;
        switch(id->type)
        {
            case ID_VAR:
                addmsg(N_EDITVAR, "risi", ID_VAR, id->name, *id->storage.i);
                break;

            case ID_FVAR:
                addmsg(N_EDITVAR, "risf", ID_FVAR, id->name, *id->storage.f);
                break;

            case ID_SVAR:
                addmsg(N_EDITVAR, "riss", ID_SVAR, id->name, *id->storage.s);
                break;
            default: return;
        }
        printvar(player1, id);
    }

    void pausegame(bool val)
    {
        if(!connected) return;
        if(!remote) server::forcepaused(val);
        else addmsg(N_PAUSEGAME, "ri", val ? 1 : 0);
    }
    ICOMMAND(pausegame, "i", (int *val), pausegame(*val > 0));
    ICOMMAND(paused, "iN$", (int *val, int *numargs, ident *id),
    { 
        if(*numargs > 0) pausegame(clampvar(id, *val, 0, 1) > 0); 
        else if(*numargs < 0) intret(gamepaused ? 1 : 0);
        else printvar(id, gamepaused ? 1 : 0); 
    });

    bool ispaused() { return gamepaused; }

    bool allowmouselook() { return !gamepaused || !remote || m_edit; }

    void changegamespeed(int val)
    {
        if(!connected) return;
        if(!remote) server::forcegamespeed(val);
        else addmsg(N_GAMESPEED, "ri", val);
    }
    ICOMMAND(gamespeed, "iN$", (int *val, int *numargs, ident *id),
    {
        if(*numargs > 0) changegamespeed(clampvar(id, *val, 10, 1000));
        else if(*numargs < 0) intret(gamespeed);
        else printvar(id, gamespeed);
    });

    int scaletime(int t) { return t*gamespeed; }

    // collect c2s messages conveniently
    vector<uchar> messages;
    int messagecn = -1, messagereliable = false;

    bool addmsg(int type, const char *fmt, ...)
    {
        if(!connected) return false;
        static uchar buf[MAXTRANS];
        ucharbuf p(buf, sizeof(buf));
        putint(p, type);
        int numi = 1, numf = 0, nums = 0, mcn = -1;
        bool reliable = false;
        if(fmt)
        {
            va_list args;
            va_start(args, fmt);
            while(*fmt) switch(*fmt++)
            {
                case 'r': reliable = true; break;
                case 'c':
                {
                    fpsent *d = va_arg(args, fpsent *);
                    mcn = !d || d == player1 ? -1 : d->clientnum;
                    break;
                }
                case 'v':
                {
                    int n = va_arg(args, int);
                    int *v = va_arg(args, int *);
                    loopi(n) putint(p, v[i]);
                    numi += n;
                    break;
                }

                case 'i':
                {
                    int n = isdigit(*fmt) ? *fmt++-'0' : 1;
                    loopi(n) putint(p, va_arg(args, int));
                    numi += n;
                    break;
                }
                case 'f':
                {
                    int n = isdigit(*fmt) ? *fmt++-'0' : 1;
                    loopi(n) putfloat(p, (float)va_arg(args, double));
                    numf += n;
                    break;
                }
                case 's': sendstring(va_arg(args, const char *), p); nums++; break;
            }
            va_end(args);
        }
        int num = nums || numf ? 0 : numi, msgsize = server::msgsizelookup(type);
        if(msgsize && num!=msgsize) { fatal("inconsistent msg size for %d (%d != %d)", type, num, msgsize); }
        if(reliable) messagereliable = true;
        if(mcn != messagecn)
        {
            static uchar mbuf[16];
            ucharbuf m(mbuf, sizeof(mbuf));
            putint(m, N_FROMAI);
            putint(m, mcn);
            messages.put(mbuf, m.length());
            messagecn = mcn;
        }
        messages.put(buf, p.length());
        return true;
    }

    void connectattempt(const char *name, const char *password, const ENetAddress &address)
    {
        copystring(connectpass, password);
    }

    void connectfail()
    {
        memset(connectpass, 0, sizeof(connectpass));
    }

    void gameconnect(bool _remote)
    {
        remote = _remote;
        if(editmode) toggleedit();
    }

    void gamedisconnect(bool cleanup)
    {
        if(remote) stopfollowing();
        ignores.setsize(0);
        connected = remote = false;
        player1->clientnum = -1;
        sessionid = 0;
        mastermode = MM_OPEN;
        messages.setsize(0);
        messagereliable = false;
        messagecn = -1;
        player1->respawn();
        player1->lifesequence = 0;
        player1->state = CS_ALIVE;
        player1->privilege = PRIV_NONE;
        sendcrc = senditemstoserver = false;
        demoplayback = false;
        gamepaused = false;
        gamespeed = 100;
        clearclients(false);
        if(cleanup)
        {
            nextmode = gamemode = INT_MAX;
            clientmap[0] = '\0';
        }
    }

    VARP(teamcolorchat, 0, 1, 1);
    const char *chatcolorname(fpsent *d) { return teamcolorchat ? teamcolorname(d, NULL) : colorname(d); }

    void toserver(char *text) { conoutf(CON_CHAT, "%s:\f0 %s", chatcolorname(player1), text); addmsg(N_TEXT, "rcs", player1, text); }
    COMMANDN(say, toserver, "C");

    void sayteam(char *text) { conoutf(CON_TEAMCHAT, "\fs\f8[team]\fr %s: \f8%s", chatcolorname(player1), text); addmsg(N_SAYTEAM, "rcs", player1, text); }
    COMMAND(sayteam, "C");

    ICOMMAND(servcmd, "C", (char *cmd), addmsg(N_SERVCMD, "rs", cmd));

    static void sendposition(fpsent *d, packetbuf &q)
    {
        putint(q, N_POS);
        putuint(q, d->clientnum);
        // 3 bits phys state, 1 bit life sequence, 2 bits move, 2 bits strafe
        uchar physstate = d->physstate | ((d->lifesequence&1)<<3) | ((d->move&3)<<4) | ((d->strafe&3)<<6);
        q.put(physstate);
        ivec o = ivec(vec(d->o.x, d->o.y, d->o.z-d->eyeheight).mul(DMF));
        uint vel = min(int(d->vel.magnitude()*DVELF), 0xFFFF), fall = min(int(d->falling.magnitude()*DVELF), 0xFFFF);
        // 3 bits position, 1 bit velocity, 3 bits falling, 1 bit material
        uint flags = 0;
        if(o.x < 0 || o.x > 0xFFFF) flags |= 1<<0;
        if(o.y < 0 || o.y > 0xFFFF) flags |= 1<<1;
        if(o.z < 0 || o.z > 0xFFFF) flags |= 1<<2;
        if(vel > 0xFF) flags |= 1<<3;
        if(fall > 0)
        {
            flags |= 1<<4;
            if(fall > 0xFF) flags |= 1<<5;
            if(d->falling.x || d->falling.y || d->falling.z > 0) flags |= 1<<6;
        }
        if((lookupmaterial(d->feetpos())&MATF_CLIP) == MAT_GAMECLIP) flags |= 1<<7;
        putuint(q, flags);
        loopk(3)
        {
            q.put(o[k]&0xFF);
            q.put((o[k]>>8)&0xFF);
            if(o[k] < 0 || o[k] > 0xFFFF) q.put((o[k]>>16)&0xFF);
        }
        uint dir = (d->yaw < 0 ? 360 + int(d->yaw)%360 : int(d->yaw)%360) + clamp(int(d->pitch+90), 0, 180)*360;
        q.put(dir&0xFF);
        q.put((dir>>8)&0xFF);
        q.put(clamp(int(d->roll+90), 0, 180));
        q.put(vel&0xFF);
        if(vel > 0xFF) q.put((vel>>8)&0xFF);
        float velyaw, velpitch;
        vectoyawpitch(d->vel, velyaw, velpitch);
        uint veldir = (velyaw < 0 ? 360 + int(velyaw)%360 : int(velyaw)%360) + clamp(int(velpitch+90), 0, 180)*360;
        q.put(veldir&0xFF);
        q.put((veldir>>8)&0xFF);
        if(fall > 0)
        {
            q.put(fall&0xFF);
            if(fall > 0xFF) q.put((fall>>8)&0xFF);
            if(d->falling.x || d->falling.y || d->falling.z > 0)
            {
                float fallyaw, fallpitch;
                vectoyawpitch(d->falling, fallyaw, fallpitch);
                uint falldir = (fallyaw < 0 ? 360 + int(fallyaw)%360 : int(fallyaw)%360) + clamp(int(fallpitch+90), 0, 180)*360;
                q.put(falldir&0xFF);
                q.put((falldir>>8)&0xFF);
            }
        }
    }

    void sendposition(fpsent *d, bool reliable)
    {
        if(d->state != CS_ALIVE && d->state != CS_EDITING) return;
        packetbuf q(100, reliable ? ENET_PACKET_FLAG_RELIABLE : 0);
        sendposition(d, q);
        sendclientpacket(q.finalize(), 0);
    }

    void sendpositions()
    {
        loopv(players)
        {
            fpsent *d = players[i];
            if((d == player1 || d->ai) && (d->state == CS_ALIVE || d->state == CS_EDITING))
            {
                packetbuf q(100);
                sendposition(d, q);
                for(int j = i+1; j < players.length(); j++)
                {
                    fpsent *d = players[j];
                    if((d == player1 || d->ai) && (d->state == CS_ALIVE || d->state == CS_EDITING))
                        sendposition(d, q);
                }
                sendclientpacket(q.finalize(), 0);
                break;
            }
        }
    }

    void sendmessages()
    {
        packetbuf p(MAXTRANS);
        if(sendcrc && !async_loading_map)
        {
            p.reliable();
            sendcrc = false;
            const char *mname = getclientmap();
            putint(p, N_MAPCRC);
            sendstring(mname, p);
            putint(p, mname[0] ? getmapcrc() : 0);
        }
        if(senditemstoserver)
        {
            if(!m_noitems || cmode!=NULL) p.reliable();
            if(!m_noitems) entities::putitems(p);
            if(cmode) cmode->senditems(p);
            senditemstoserver = false;
        }
        if(messages.length())
        {
            p.put(messages.getbuf(), messages.length());
            messages.setsize(0);
            if(messagereliable) p.reliable();
            messagereliable = false;
            messagecn = -1;
        }
        if(totalmillis-lastping>250)
        {
            putint(p, N_PING);
            putint(p, totalmillis);
            lastping = totalmillis;
        }
        sendclientpacket(p.finalize(), 1);
    }

    void c2sinfo(bool force) // send update to the server
    {
        static int lastupdate = -1000;
        if(totalmillis - lastupdate < 33 && !force) return; // don't update faster than 30fps
        lastupdate = totalmillis;
        sendpositions();
        sendmessages();
        flushclient();
    }

    void sendintro()
    {
        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
        putint(p, N_CONNECT);
        sendstring(player1->name, p);
        putint(p, player1->playermodel);
        string hash = "";
        if(connectpass[0])
        {
            server::hashpassword(player1->clientnum, sessionid, connectpass, hash);
            memset(connectpass, 0, sizeof(connectpass));
        }
        sendstring(hash, p);
        authkey *a = servauth[0] && autoauth ? findauthkey(servauth) : NULL;
        if(a)
        {
            a->lastauth = lastmillis;
            sendstring(a->desc, p);
            sendstring(a->name, p);
        }
        else
        {
            sendstring("", p);
            sendstring("", p);
        }
        sendclientpacket(p.finalize(), 1);
    }

    void updatepos(fpsent *d)
    {
        // update the position of other clients in the game in our world
        // don't care if he's in the scenery or other players,
        // just don't overlap with our client

        const float r = player1->radius+d->radius;
        const float dx = player1->o.x-d->o.x;
        const float dy = player1->o.y-d->o.y;
        const float dz = player1->o.z-d->o.z;
        const float rz = player1->aboveeye+d->eyeheight;
        const float fx = (float)fabs(dx), fy = (float)fabs(dy), fz = (float)fabs(dz);
        if(fx<r && fy<r && fz<rz && player1->state!=CS_SPECTATOR && d->state!=CS_DEAD)
        {
            if(fx<fy) d->o.y += dy<0 ? r-fy : -(r-fy);  // push aside
            else      d->o.x += dx<0 ? r-fx : -(r-fx);
        }
        int lagtime = totalmillis-d->lastupdate;
        if(lagtime)
        {
            if(d->state!=CS_SPAWNING && d->lastupdate) d->plag = (d->plag*5+lagtime)/6;
            d->lastupdate = totalmillis;
        }
    }

    void parsepositions(ucharbuf &p)
    {
        int type;
        while(p.remaining()) switch(type = getint(p))
        {
            case N_DEMOPACKET: break;
            case N_POS:                        // position of another client
            {
                int cn = getuint(p), physstate = p.get(), flags = getuint(p);
                vec o, vel, falling;
                float yaw, pitch, roll;
                loopk(3)
                {
                    int n = p.get(); n |= p.get()<<8; if(flags&(1<<k)) { n |= p.get()<<16; if(n&0x800000) n |= ~0U<<24; }
                    o[k] = n/DMF;
                }
                int dir = p.get(); dir |= p.get()<<8;
                yaw = dir%360;
                pitch = clamp(dir/360, 0, 180)-90;
                roll = clamp(int(p.get()), 0, 180)-90;
                int mag = p.get(); if(flags&(1<<3)) mag |= p.get()<<8;
                dir = p.get(); dir |= p.get()<<8;
                vecfromyawpitch(dir%360, clamp(dir/360, 0, 180)-90, 1, 0, vel);
                vel.mul(mag/DVELF);
                if(flags&(1<<4))
                {
                    mag = p.get(); if(flags&(1<<5)) mag |= p.get()<<8;
                    if(flags&(1<<6))
                    {
                        dir = p.get(); dir |= p.get()<<8;
                        vecfromyawpitch(dir%360, clamp(dir/360, 0, 180)-90, 1, 0, falling);
                    }
                    else falling = vec(0, 0, -1);
                    falling.mul(mag/DVELF);
                }
                else falling = vec(0, 0, 0);
                int seqcolor = (physstate>>3)&1;
                fpsent *d = getclient(cn);
                if(!d || d->lifesequence < 0 || seqcolor!=(d->lifesequence&1) || d->state==CS_DEAD) continue;
                float oldyaw = d->yaw, oldpitch = d->pitch, oldroll = d->roll;
                d->yaw = yaw;
                d->pitch = pitch;
                d->roll = roll;
                d->move = (physstate>>4)&2 ? -1 : (physstate>>4)&1;
                d->strafe = (physstate>>6)&2 ? -1 : (physstate>>6)&1;
                vec oldpos(d->o);
                d->o = o;
                d->o.z += d->eyeheight;
                d->vel = vel;
                d->falling = falling;
                d->physstate = physstate&7;
                updatephysstate(d);
                updatepos(d);
                if(smoothmove && d->smoothmillis>=0 && oldpos.dist(d->o) < smoothdist)
                {
                    d->newpos = d->o;
                    d->newyaw = d->yaw;
                    d->newpitch = d->pitch;
                    d->newroll = d->roll;
                    d->o = oldpos;
                    d->yaw = oldyaw;
                    d->pitch = oldpitch;
                    d->roll = oldroll;
                    (d->deltapos = oldpos).sub(d->newpos);
                    d->deltayaw = oldyaw - d->newyaw;
                    if(d->deltayaw > 180) d->deltayaw -= 360;
                    else if(d->deltayaw < -180) d->deltayaw += 360;
                    d->deltapitch = oldpitch - d->newpitch;
                    d->deltaroll = oldroll - d->newroll;
                    d->smoothmillis = lastmillis;
                }
                else d->smoothmillis = 0;
                if(d->state==CS_LAGGED || d->state==CS_SPAWNING) d->state = CS_ALIVE;
                break;
            }

            case N_TELEPORT:
            {
                int cn = getint(p), tp = getint(p), td = getint(p);
                fpsent *d = getclient(cn);
                if(!d || d->lifesequence < 0 || d->state==CS_DEAD) continue;
                entities::teleporteffects(d, tp, td, false);
                break;
            }

            case N_JUMPPAD:
            {
                int cn = getint(p), jp = getint(p);
                fpsent *d = getclient(cn);
                if(!d || d->lifesequence < 0 || d->state==CS_DEAD) continue;
                entities::jumppadeffects(d, jp, false);
                break;
            }

            default:
                neterr("type");
                return;
        }
    }

    void parsestate(fpsent *d, ucharbuf &p, bool resume = false)
    {
        if(!d) { static fpsent dummy; d = &dummy; }
        if(resume)
        {
            if(d==player1) getint(p);
            else d->state = getint(p);
            d->frags = getint(p);
            d->flags = getint(p);
            d->deaths = getint(p);
            if(d==player1) getint(p);
            else d->quadmillis = getint(p);
        }
        d->lifesequence = getint(p);
        d->health = getint(p);
        d->maxhealth = getint(p);
        d->armour = getint(p);
        d->armourtype = getint(p);
        if(resume && d==player1)
        {
            getint(p);
            loopi(GUN_PISTOL-GUN_SG+1) getint(p);
        }
        else
        {
            int gun = getint(p);
            d->gunselect = clamp(gun, int(GUN_FIST), int(GUN_PISTOL));
            loopi(GUN_PISTOL-GUN_SG+1) d->ammo[GUN_SG+i] = getint(p);
        }
    }

    extern int deathscore;

    void parsemessages(int cn, fpsent *d, ucharbuf &p)
    {
        static char text[MAXTRANS];
        int type;
        bool mapchanged = false, demopacket = false;

        while(p.remaining()) {
            int type = getint(p);

            if (false) {
                conoutf("server -> client %s", gettype(type));
            }
            switch(type)
        {
            case N_DEMOPACKET: demopacket = true; break;

            case N_SERVINFO:                   // welcome messsage from the server
                {
                    int mycn = getint(p), prot = getint(p);
                    if(prot!=PROTOCOL_VERSION)
                    {
                        conoutf(CON_ERROR, "you are using a different game protocol (you: %d, server: %d)", PROTOCOL_VERSION, prot);
                        disconnect();
                        return;
                    }
                    sessionid = getint(p);
                    player1->clientnum = mycn;      // we are now connected
                    if(getint(p) > 0) conoutf("this server is password protected");
                    getstring(servinfo, p, sizeof(servinfo));
                    getstring(servauth, p, sizeof(servauth));
                    sendintro();
                    break;
                }

            case N_WELCOME:
            {
                connected = true;
                notifywelcome();
                break;
            }

            case N_PAUSEGAME:
            {
                bool val = getint(p) > 0;
                int cn = getint(p);
                fpsent *a = cn >= 0 ? getclient(cn) : NULL;
                if(!demopacket)
                {
                    gamepaused = val;
                    player1->attacking = false;
                }
                if(a) conoutf("%s %s the game", colorname(a), val ? "paused" : "resumed"); 
                else conoutf("game is %s", val ? "paused" : "resumed");
                break;
            }

            case N_GAMESPEED:
            {
                int val = clamp(getint(p), 10, 1000), cn = getint(p);
                fpsent *a = cn >= 0 ? getclient(cn) : NULL;
                if(!demopacket) gamespeed = val;
                extern int slowmosp;
                if(m_sp && slowmosp) break;
                if(a) conoutf("%s set gamespeed to %d", colorname(a), val);
                else conoutf("gamespeed is %d", val);
                break;
            }
                
            case N_CLIENT:
                {
                    int cn = getint(p), len = getuint(p);
                    ucharbuf q = p.subbuf(len);
                    parsemessages(cn, getclient(cn), q);
                    break;
                }

            case N_SOUND:
                if(!d) return;
                playsound(getint(p), &d->o);
                break;

            case N_TEXT:
            {
                if(!d) return;
                getstring(text, p);
                filtertext(text, text, true, true);
                if(isignored(d->clientnum)) break;
                if(d->state!=CS_DEAD && d->state!=CS_SPECTATOR)
                    particle_textcopy(d->abovehead(), text, PART_TEXT, 2000, 0x32FF64, 4.0f, -8);
                conoutf(CON_CHAT, "%s:\f0 %s", chatcolorname(d), text);
                break;
            }

            case N_SAYTEAM:
            {
                int tcn = getint(p);
                fpsent *t = getclient(tcn);
                getstring(text, p);
                filtertext(text, text, true, true);
                if(!t || isignored(t->clientnum)) break;
                if(t->state!=CS_DEAD && t->state!=CS_SPECTATOR)
                    particle_textcopy(t->abovehead(), text, PART_TEXT, 2000, 0x6496FF, 4.0f, -8);
                conoutf(CON_TEAMCHAT, "\fs\f8[team]\fr %s: \f8%s", chatcolorname(t), text);
                break;
            }

            case N_MAPCHANGE:
                getstring(text, p);
                filtertext(text, text, false);
                fixmapname(text);
                changemapserv(text, getint(p));
                mapchanged = true;
                if(getint(p)) entities::spawnitems();
                else senditemstoserver = false;
                break;

            case N_FORCEDEATH:
                {
                    int cn = getint(p);
                    fpsent *d = cn==player1->clientnum ? player1 : newclient(cn);
                    if(!d) break;
                    if(d==player1)
                    {
                        if(editmode) toggleedit();
                        stopfollowing();
                        if(deathscore) showscores(true);
                    }
                    else d->resetinterp();
                    d->state = CS_DEAD;
                    break;
                }

            case N_ITEMLIST:
                {
                    int n;
                    while((n = getint(p))>=0 && !p.overread())
                    {
#if __EMSCRIPTEN__
                        // XXX does this break anything?
                        entities::setspawn(n, true);
#else
                        if(mapchanged) entities::setspawn(n, true);
#endif
                        getint(p); // type
                    }
                    break;
                }

            case N_INITCLIENT:            // another client either connected or changed name/team
                {
                    int cn = getint(p);
                    fpsent *d = newclient(cn);
                    if(!d)
                    {
                        getstring(text, p);
                        getstring(text, p);
                        getint(p);
                        break;
                    }
                    getstring(text, p);
                    filtertext(text, text, false, MAXNAMELEN);
                    if(!text[0]) copystring(text, "unnamed");
                    if(d->name[0])          // already connected
                    {
                        if(strcmp(d->name, text) && !isignored(d->clientnum))
                            conoutf("%s is now known as %s", colorname(d), colorname(d, text));
                    }
                    else                    // new client
                    {
                        conoutf("connected: %s", colorname(d, text));
                        if(needclipboard >= 0) needclipboard++;
                    }
                    copystring(d->name, text, MAXNAMELEN+1);
                    getstring(text, p);
                    filtertext(d->team, text, false, MAXTEAMLEN);
                    d->playermodel = getint(p);
                    break;
                }

            case N_SWITCHNAME:
                getstring(text, p);
                if(d)
                {
                    filtertext(text, text, false, false, MAXNAMELEN);
                    if(!text[0]) copystring(text, "unnamed");
                    if(strcmp(text, d->name))
                    {
                        if(!isignored(d->clientnum)) conoutf("%s is now known as %s", colorname(d), colorname(d, text));
                        copystring(d->name, text, MAXNAMELEN+1);
                    }
                }
                break;

            case N_SWITCHMODEL:
                {
                    int model = getint(p);
                    if(d)
                    {
                        d->playermodel = model;
                        if(d->ragdoll) cleanragdoll(d);
                    }
                    break;
                }

            case N_CDIS:
                clientdisconnected(getint(p));
                break;

            case N_SPAWN:
                {
                    if(d)
                    {
                        if(d->state==CS_DEAD && d->lastpain) saveragdoll(d);
                        d->respawn();
                    }
                    parsestate(d, p);
                    if(!d) break;
                    d->state = CS_SPAWNING;
                    if(player1->state==CS_SPECTATOR && following==d->clientnum)
                        lasthit = 0;
                    break;
                }

            case N_SPAWNSTATE:
            {
                int scn = getint(p);
                fpsent *s = getclient(scn);
                if(!s) { parsestate(NULL, p); break; }
                if(s->state==CS_DEAD && s->lastpain) saveragdoll(s);
                if(s==player1)
                {
                    if(editmode) toggleedit();
                    stopfollowing();
                }
                s->respawn();
                parsestate(s, p);
                s->state = CS_ALIVE;
                pickgamespawn(s);
                if(s == player1)
                {
                    showscores(false);
                    lasthit = 0;
                }
                if(cmode) cmode->respawned(s);
				ai::spawned(s);
                addmsg(N_SPAWN, "rcii", s, s->lifesequence, s->gunselect);
                break;
            }

            case N_SHOTFX:
                {
                    int scn = getint(p), gun = getint(p), id = getint(p);
                    vec from, to;
                    loopk(3) from[k] = getint(p)/DMF;
                    loopk(3) to[k] = getint(p)/DMF;
                    fpsent *s = getclient(scn);
                    if(!s) break;
                    if(gun>GUN_FIST && gun<=GUN_PISTOL && s->ammo[gun]) s->ammo[gun]--;
                    s->gunselect = clamp(gun, (int)GUN_FIST, (int)GUN_PISTOL);
                    s->gunwait = guns[s->gunselect].attackdelay;
                    int prevaction = s->lastaction;
                    s->lastaction = lastmillis;
                    s->lastattackgun = s->gunselect;
                    shoteffects(s->gunselect, from, to, s, false, id, prevaction);
                    break;
                }

            case N_EXPLODEFX:
                {
                    int ecn = getint(p), gun = getint(p), id = getint(p);
                    fpsent *e = getclient(ecn);
                    if(!e) break;
                    explodeeffects(gun, e, false, id);
                    break;
                }
            case N_DAMAGE:
                {
                    int tcn = getint(p),
                        acn = getint(p),
                        damage = getint(p),
                        armour = getint(p),
                        health = getint(p);
                    fpsent *target = getclient(tcn),
                           *actor = getclient(acn);
                    if(!target || !actor) break;
                    target->armour = armour;
                    target->health = health;
                    if(target->state == CS_ALIVE && actor != player1) target->lastpain = lastmillis;
                    damaged(damage, target, actor, false);
                    break;
                }

            case N_HITPUSH:
                {
                    int tcn = getint(p), gun = getint(p), damage = getint(p);
                    fpsent *target = getclient(tcn);
                    vec dir;
                    loopk(3) dir[k] = getint(p)/DNF;
                    if(target) target->hitpush(damage * (target->health<=0 ? deadpush : 1), dir, NULL, gun);
                    break;
                }

            case N_DIED:
            {
                int vcn = getint(p), acn = getint(p), frags = getint(p), tfrags = getint(p);
                fpsent *victim = getclient(vcn),
                       *actor = getclient(acn);
                if(!actor) break;
                actor->frags = frags;
                if(m_teammode) setteaminfo(actor->team, tfrags);
                extern int hidefrags;
                if(actor!=player1 && (!cmode || !cmode->hidefrags() || !hidefrags))
                {
                    defformatstring(ds, "%d", actor->frags);
                    particle_textcopy(actor->abovehead(), ds, PART_TEXT, 2000, 0x32FF64, 4.0f, -8);
                }
                if(!victim) break;
                killed(victim, actor);
                break;
            }

            case N_TEAMINFO:
                for(;;)
                {
                    getstring(text, p);
                    if(p.overread() || !text[0]) break;
                    int frags = getint(p);
                    if(p.overread()) break;
                    if(m_teammode) setteaminfo(text, frags);
                }
                break;

            case N_GUNSELECT:
                {
                    if(!d) return;
                    int gun = getint(p);
                    d->gunselect = clamp(gun, int(GUN_FIST), int(GUN_PISTOL));
                    playsound(S_WEAPLOAD, &d->o);
                    break;
                }

            case N_TAUNT:
                {
                    if(!d) return;
                    d->lasttaunt = lastmillis;
                    break;
                }

            case N_RESUME:
                {
                    for(;;)
                    {
                        int cn = getint(p);
                        if(p.overread() || cn<0) break;
                        fpsent *d = (cn == player1->clientnum ? player1 : newclient(cn));
                        parsestate(d, p, true);
                    }
                    break;
                }

            case N_ITEMSPAWN:
            {
                int i = getint(p);
                if(!entities::ents.inrange(i)) break;
                entities::setspawn(i, true);
                ai::itemspawned(i);
                playsound(S_ITEMSPAWN, &entities::ents[i]->o, NULL, 0, 0, 0, -1, 0, 1500);
                #if 0
                const char *name = entities::itemname(i);
                if(name) particle_text(entities::ents[i]->o, name, PART_TEXT, 2000, 0x32FF64, 4.0f, -8);
                #endif
                int icon = entities::itemicon(i);
                if(icon >= 0) particle_icon(vec(0.0f, 0.0f, 4.0f).add(entities::ents[i]->o), icon%4, icon/4, PART_HUD_ICON, 2000, 0xFFFFFF, 2.0f, -8);
                break;
            }

            case N_ITEMACC:            // server acknowledges that I picked up this item
            {
                int i = getint(p), cn = getint(p);
                if(cn >= 0)
                {
                    fpsent *d = getclient(cn);
                    entities::pickupeffects(i, d);
                }
                else entities::setspawn(i, true);
                break;
            }

            case N_CLIPBOARD:
            {
                int cn = getint(p), unpacklen = getint(p), packlen = getint(p);
                fpsent *d = getclient(cn);
                ucharbuf q = p.subbuf(max(packlen, 0));
                if(d) unpackeditinfo(d->edit, q.buf, q.maxlen, unpacklen);
                break;
            }
            case N_UNDO:
            case N_REDO:
            {
                int cn = getint(p), unpacklen = getint(p), packlen = getint(p);
                fpsent *d = getclient(cn);
                ucharbuf q = p.subbuf(max(packlen, 0));
                if(d) unpackundo(q.buf, q.maxlen, unpacklen);
                break;
            }

            case N_EDITF:              // coop editing messages
            case N_EDITT:
            case N_EDITM:
            case N_FLIP:
            case N_COPY:
            case N_PASTE:
            case N_ROTATE:
            case N_REPLACE:
            case N_DELCUBE:
            case N_EDITVSLOT:
            {
                if(!d) return;
                selinfo sel;
                sel.o.x = getint(p); sel.o.y = getint(p); sel.o.z = getint(p);
                sel.s.x = getint(p); sel.s.y = getint(p); sel.s.z = getint(p);
                sel.grid = getint(p); sel.orient = getint(p);
                sel.cx = getint(p); sel.cxs = getint(p); sel.cy = getint(p), sel.cys = getint(p);
                sel.corner = getint(p);
                switch(type)
                {
                    case N_EDITF: { int dir = getint(p), mode = getint(p); if(sel.validate()) mpeditface(dir, mode, sel, false); break; }
                    case N_EDITT:
                    {
                        int tex = getint(p),
                            allfaces = getint(p);
                        if(p.remaining() < 2) return;
                        int extra = lilswap(*(const ushort *)p.pad(2));
                        if(p.remaining() < extra) return;
                        ucharbuf ebuf = p.subbuf(extra);
                        if(sel.validate()) mpedittex(tex, allfaces, sel, ebuf);
                        break;
                    }
                    case N_EDITM: { int mat = getint(p), filter = getint(p); if(sel.validate()) mpeditmat(mat, filter, sel, false); break; }
                    case N_FLIP: if(sel.validate()) mpflip(sel, false); break;
                    case N_COPY: if(d && sel.validate()) mpcopy(d->edit, sel, false); break;
                    case N_PASTE: if(d && sel.validate()) mppaste(d->edit, sel, false); break;
                    case N_ROTATE: { int dir = getint(p); if(sel.validate()) mprotate(dir, sel, false); break; }
                    case N_REPLACE:
                    {
                        int oldtex = getint(p),
                            newtex = getint(p),
                            insel = getint(p);
                        if(p.remaining() < 2) return;
                        int extra = lilswap(*(const ushort *)p.pad(2));
                        if(p.remaining() < extra) return;
                        ucharbuf ebuf = p.subbuf(extra);
                        if(sel.validate()) mpreplacetex(oldtex, newtex, insel>0, sel, ebuf);
                        break;
                    }
                    case N_DELCUBE: if(sel.validate()) mpdelcube(sel, false); break;
                    case N_EDITVSLOT:
                    {
                        int delta = getint(p),
                            allfaces = getint(p);
                        if(p.remaining() < 2) return;
                        int extra = lilswap(*(const ushort *)p.pad(2));
                        if(p.remaining() < extra) return;
                        ucharbuf ebuf = p.subbuf(extra);
                        if(sel.validate()) mpeditvslot(delta, allfaces, sel, ebuf);
                        break;
                    }
                }
                break;
            }

            case N_REMIP:
                {
                    if(!d) return;
                    conoutf("%s remipped", colorname(d));
                    mpremip(false);
                    break;
                }
            case N_EDITENT:            // coop edit of ent
                {
                    if(!d) return;
                    int i = getint(p);
                    float x = getint(p)/DMF, y = getint(p)/DMF, z = getint(p)/DMF;
                    int type = getint(p);
                    int attr1 = getint(p), attr2 = getint(p), attr3 = getint(p), attr4 = getint(p), attr5 = getint(p);

                    mpeditent(i, vec(x, y, z), type, attr1, attr2, attr3, attr4, attr5, false);
                    break;
                }

            case N_EDITVAR:
            {
                if(!d) return;
                int type = getint(p);
                getstring(text, p);
                string name;
                filtertext(name, text, false);
                ident *id = getident(name);
                switch(type)
                {
                    case ID_VAR:
                    {
                        int val = getint(p);
                        if(id && id->flags&IDF_OVERRIDE && !(id->flags&IDF_READONLY)) setvar(name, val);
                        break;
                    }
                    case ID_FVAR:
                    {
                        float val = getfloat(p);
                        if(id && id->flags&IDF_OVERRIDE && !(id->flags&IDF_READONLY)) setfvar(name, val);
                        break;
                    }
                    case ID_SVAR:
                    {
                        getstring(text, p);
                        if(id && id->flags&IDF_OVERRIDE && !(id->flags&IDF_READONLY)) setsvar(name, text);
                        break;
                    }
                }
                printvar(d, id);
                break;
            }

            case N_PONG:
                addmsg(N_CLIENTPING, "i", player1->ping = (player1->ping*5+totalmillis-getint(p))/6);
                break;

            case N_CLIENTPING:
                if(!d) return;
                d->ping = getint(p);
                break;

            case N_TIMEUP:
                timeupdate(getint(p));
                break;

            case N_SERVMSG:
                getstring(text, p);
                conoutf("%s", text);
                break;

            case N_SENDDEMOLIST:
            {
                int demos = getint(p);
                if(demos <= 0) conoutf("no demos available");
                else loopi(demos)
                {
                    getstring(text, p);
                    if(p.overread()) break;
                    conoutf("%d. %s", i+1, text);
                }
                break;
            }

            case N_DEMOPLAYBACK:
            {
                int on = getint(p);
                if(on) player1->state = CS_SPECTATOR;
                else clearclients();
                demoplayback = on!=0;
                player1->clientnum = getint(p);
                gamepaused = false;
                execident(on ? "demostart" : "demoend");
                break;
            }

            case N_CURRENTMASTER:
                {
                    int mm = getint(p), mn;
                    loopv(players) players[i]->privilege = PRIV_NONE;
                    while((mn = getint(p))>=0 && !p.overread())
                    {
                        fpsent *m = mn==player1->clientnum ? player1 : newclient(mn);
                        int priv = getint(p);
                        if(m) m->privilege = priv;
                    }
                    if(mm != mastermode)
                    {
                        mastermode = mm;
                        conoutf("mastermode is %s (%d)", server::mastermodename(mastermode), mastermode);
                    }
                    break;
                }

            case N_MASTERMODE:
                {
                    mastermode = getint(p);
                    conoutf("mastermode is %s (%d)", server::mastermodename(mastermode), mastermode);
                    break;
                }

            case N_EDITMODE:
                {
                    int val = getint(p);
                    if(!d) break;
                    if(val)
                    {
                        d->editstate = d->state;
                        d->state = CS_EDITING;
                    }
                    else
                    {
                        d->state = d->editstate;
                        if(d->state==CS_DEAD) deathstate(d, true);
                    }
                    break;
                }

            case N_SPECTATOR:
                {
                    int sn = getint(p), val = getint(p);
                    fpsent *s;
                    if(sn==player1->clientnum)
                    {
                        s = player1;
                        if(val && remote && !player1->privilege) senditemstoserver = false;
                    }
                    else s = newclient(sn);
                    if(!s) return;
                    if(val)
                    {
                        if(s==player1)
                        {
                            if(editmode) toggleedit();
                            if(s->state==CS_DEAD) showscores(false);
                            disablezoom();
                        }
                        s->state = CS_SPECTATOR;
                    }
                    else if(s->state==CS_SPECTATOR)
                    {
                        if(s==player1) stopfollowing();
                        deathstate(s, true);
                    }
                    break;
                }

            case N_SETTEAM:
            {
                int wn = getint(p);
                getstring(text, p);
                int reason = getint(p);
                fpsent *w = getclient(wn);
                if(!w) return;
                filtertext(w->team, text, false, false, MAXTEAMLEN);
                static const char * const fmt[2] = { "%s switched to team %s", "%s forced to team %s"};
                if(reason >= 0 && size_t(reason) < sizeof(fmt)/sizeof(fmt[0]))
                    conoutf(fmt[reason], colorname(w), w->team);
                break;
            }

#define PARSEMESSAGES 1
#include "capture.h"
#include "ctf.h"
#include "collect.h"
#undef PARSEMESSAGES

            case N_ANNOUNCE:
            {
                int t = getint(p);
                if     (t==I_QUAD)  { playsound(S_V_QUAD10, NULL, NULL, 0, 0, 0, -1, 0, 3000);  conoutf(CON_GAMEINFO, "\f2quad damage will spawn in 10 seconds!"); }
                else if(t==I_BOOST) { playsound(S_V_BOOST10, NULL, NULL, 0, 0, 0, -1, 0, 3000); conoutf(CON_GAMEINFO, "\f2health boost will spawn in 10 seconds!"); }
                break;
            }

            case N_NEWMAP:
                {
                    int size = getint(p);
                    if(size>=0) emptymap(size, true, NULL);
                    else enlargemap(true);
                    if(d && d!=player1)
                    {
                        int newsize = 0;
                        while(1<<newsize < getworldsize()) newsize++;
                        conoutf(size>=0 ? "%s started a new map of size %d" : "%s enlarged the map to size %d", colorname(d), newsize);
                    }
                    break;
                }

            case N_REQAUTH:
                {
                    getstring(text, p);
                    if(autoauth && text[0] && tryauth(text)) conoutf("server requested authkey \"%s\"", text);
                    break;
                }

            case N_AUTHCHAL:
            {
                getstring(text, p);
                authkey *a = findauthkey(text);
                uint id = (uint)getint(p);
                getstring(text, p);
                vector<char> buf;
                if(a && a->lastauth && lastmillis - a->lastauth < 60*1000 && answerchallenge(a->key, text, buf))
                {
                    //conoutf(CON_DEBUG, "answering %u, challenge %s with %s", id, text, buf.getbuf());
                    packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
                    putint(p, N_AUTHANS);
                    sendstring(a->desc, p);
                    putint(p, id);
                    sendstring(buf.getbuf(), p);
                    sendclientpacket(p.finalize(), 1);
                }
                break;
            }


            case N_INITAI:
            {
                int bn = getint(p), on = getint(p), at = getint(p), sk = clamp(getint(p), 1, 101), pm = getint(p);
                string name, team;
                getstring(text, p);
                filtertext(name, text, false, false, MAXNAMELEN);
                getstring(text, p);
                filtertext(team, text, false, false, MAXTEAMLEN);
                fpsent *b = newclient(bn);
                if(!b) break;
                ai::init(b, at, on, sk, bn, pm, name, team);
                break;
            }


            case N_SERVCMD:
                getstring(text, p);
                break;

            default:
                conoutf("got unknown message type=%d", type);
                neterr("type", cn < 0);
                return;
        }
        }
    }

    struct demoreq
    {
        int tag;
        string name;
    };
    vector<demoreq> demoreqs;
    enum { MAXDEMOREQS = 7 };
    static int lastdemoreq = 0;

    void receivefile(packetbuf &p)
    {
        int type;
        while(p.remaining()) switch(type = getint(p))
        {
            case N_DEMOPACKET: return;
            case N_SENDDEMO:
            {
                string fname;
                fname[0] = '\0';
                int tag = getint(p);
                loopv(demoreqs) if(demoreqs[i].tag == tag)
                {
                    copystring(fname, demoreqs[i].name);
                    demoreqs.remove(i);
                    break;
                }
                if(!fname[0])
                {
                    time_t t = time(NULL);
                    size_t len = strftime(fname, sizeof(fname), "%Y-%m-%d_%H.%M.%S", localtime(&t));
                    fname[min(len, sizeof(fname)-1)] = '\0';
                }
                int len = strlen(fname);
                if(len < 4 || strcasecmp(&fname[len-4], ".dmo")) concatstring(fname, ".dmo");
                stream *demo = NULL;
                if(const char *buf = server::getdemofile(fname, true)) demo = openrawfile(buf, "wb");
                if(!demo) demo = openrawfile(fname, "wb");
                if(!demo) return;
                conoutf("received demo \"%s\"", fname);
                ucharbuf b = p.subbuf(p.remaining());
                demo->write(b.buf, b.maxlen);
                delete demo;
                break;
            }

            case N_SENDMAP:
            {
                if(!m_edit) return;
                string oldname;
                copystring(oldname, getclientmap());
                defformatstring(mname, "getmap_%d", lastmillis);
                defformatstring(fname, "packages/base/%s.ogz", mname);
                stream *map = openrawfile(path(fname), "wb");
                if(!map) return;
                ucharbuf b = p.subbuf(p.remaining());
                map->write(b.buf, b.maxlen);
                delete map;

#if __EMSCRIPTEN__
				if (oldname[0]) {
					EM_ASM({
						return Module.assets.receiveMap(UTF8ToString($0), UTF8ToString($1))
					}, mname, oldname);
				} else {
					EM_ASM({
						return Module.assets.receiveMap(UTF8ToString($0), "")
					}, mname);
				}
#else
				if(really_load_world(mname, oldname[0] ? oldname : NULL))
					entities::spawnitems(true);
				// Doesn't matter in Emscripten since FS is in-memory anyway
                remove(findfile(fname, "rb"));
#endif
                break;
            }
        }
    }

    void parsepacketclient(int chan, packetbuf &p)   // processes any updates from the server
    {
        if(p.packet->flags&ENET_PACKET_FLAG_UNSEQUENCED) return;
        switch(chan)
        {
            case 0:
                parsepositions(p);
                break;

            case 1:
                parsemessages(-1, NULL, p);
                break;

            case 2:
                receivefile(p);
                break;
        }
    }

    void getmap()
    {
        if(!m_edit) { conoutf(CON_ERROR, "\"getmap\" only works in coop edit mode"); return; }
        conoutf("getting map...");
        addmsg(N_GETMAP, "r");
    }
    COMMAND(getmap, "");

    void spawnitems()
    {
        entities::spawnitems(true);
    }
    COMMAND(spawnitems, "");

    void stopdemo()
    {
        if(remote)
        {
            if(player1->privilege<PRIV_MASTER) return;
            addmsg(N_STOPDEMO, "r");
        }
        else server::stopdemo();
    }
    COMMAND(stopdemo, "");

    void recorddemo(int val)
    {
        if(remote && player1->privilege<PRIV_MASTER) return;
        addmsg(N_RECORDDEMO, "ri", val);
    }
    ICOMMAND(recorddemo, "i", (int *val), recorddemo(*val));

    void cleardemos(int val)
    {
        if(remote && player1->privilege<PRIV_MASTER) return;
        addmsg(N_CLEARDEMOS, "ri", val);
    }
    ICOMMAND(cleardemos, "i", (int *val), cleardemos(*val));

    void getdemo(char *val, char *name)
    {
        int i = 0;
        if(isdigit(val[0]) || name[0]) i = parseint(val);
        else name = val;
        if(i<=0) conoutf("getting demo...");
        else conoutf("getting demo %d...", i);
        ++lastdemoreq;
        if(name[0])
        {
            if(demoreqs.length() >= MAXDEMOREQS) demoreqs.remove(0);
            demoreq &r = demoreqs.add();
            r.tag = lastdemoreq;
            copystring(r.name, name);
        }
        addmsg(N_GETDEMO, "rii", i, lastdemoreq);
    }
    ICOMMAND(getdemo, "ss", (char *val, char *name), getdemo(val, name));

    void listdemos()
    {
        conoutf("listing demos...");
        addmsg(N_LISTDEMOS, "r");
    }
    COMMAND(listdemos, "");

    void sendmap()
    {
        if(!m_edit || (player1->state==CS_SPECTATOR && remote && !player1->privilege)) { conoutf(CON_ERROR, "\"sendmap\" only works in coop edit mode"); return; }
        conoutf("sending map...");
        defformatstring(mname, "sendmap_%d", lastmillis);
        save_world(mname, true);
        defformatstring(fname, "packages/base/%s.ogz", mname);
        stream *map = openrawfile(path(fname), "rb");
        if(map)
        {
            stream::offset len = map->size();
            if(len > 4*1024*1024) conoutf(CON_ERROR, "map is too large");
            else if(len <= 0) conoutf(CON_ERROR, "could not read map");
            else
            {
                sendfile(-1, 2, map);
                if(needclipboard >= 0) needclipboard++;
            }
            delete map;
        }
        else conoutf(CON_ERROR, "could not read map");
        remove(findfile(fname, "rb"));
    }
    COMMAND(sendmap, "");

    void gotoplayer(const char *arg)
    {
        if(player1->state!=CS_SPECTATOR && player1->state!=CS_EDITING) return;
        int i = parseplayer(arg);
        if(i>=0)
        {
            fpsent *d = getclient(i);
            if(!d || d==player1) return;
            player1->o = d->o;
            vec dir;
            vecfromyawpitch(player1->yaw, player1->pitch, 1, 0, dir);
            player1->o.add(dir.mul(-32));
            player1->resetinterp();
        }
    }
    COMMANDN(goto, gotoplayer, "s");

    void gotosel()
    {
        if(player1->state!=CS_EDITING) return;
        player1->o = getselpos();
        vec dir;
        vecfromyawpitch(player1->yaw, player1->pitch, 1, 0, dir);
        player1->o.add(dir.mul(-32));
        player1->resetinterp();
    }
    COMMAND(gotosel, "");
}

