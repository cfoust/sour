#include "../mod/QServ.h"

int count = 0;
int msgcount[128];

namespace game {
    void parseoptions(vector<const char *> &args)
    {
        loopv(args)
        if(!server::serveroption(args[i]))
            conoutf(CON_ERROR, "No such command \"%s\"", args[i]);
    }
    const char *gameident() { return "fps"; }

    void _changemap(const char *s, int mode) {
        server::_changemap(s, mode);
    }
}

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

void out(int type, const char *fmt, ...)
{
    char msg[1000];
    va_list list;
    va_start(list,fmt);
    vsnprintf(msg,1000,fmt,list);
    va_end(list);

    switch(type)
    {
        case ECHO_ALL:
        {
            server::sendservmsg(msg);
            puts(msg);
            break;
        }
        case ECHO_IRC:
        {
            break;
        }
        case ECHO_CONSOLE:
        {
            puts(msg);
            break;
        }
        case ECHO_SERV:
        {
            server::sendservmsg(msg);
            break;
        }
        case ECHO_NOCOLOR:
        {
            puts(msg);
            break;
        }
        default:
            break;
    }
}

extern ENetAddress masteraddress;

//Main server namespace
namespace server {

    bool duplicatename(clientinfo *ci, char *name) {
        if(!name) name = ci->name;
        loopv(clients) if(clients[i]!=ci && !strcmp(name, clients[i]->name)) return true;
        return false;
    }

    const char *colorname(clientinfo *ci) {
        char *name = NULL;

        if(!name) name = ci->name;
        if(name[0] && !duplicatename(ci, name) && ci->state.aitype == AI_NONE) return name;
        static string cname[3];
        static int cidx = 0;
        cidx = (cidx+1)%3;
        formatstring(cname[cidx])(ci->state.aitype == AI_NONE ? "%s \fs\f5(%d)\fr" : "%s \fs\f5[%d]\fr", name, ci->clientnum);
        return cname[cidx];
    }

    vector<uint> allowedips;
    vector<ban> bannedips;

    void clearbans() {
        bannedips.shrink(0);
        out(ECHO_SERV, "Server bans \f0cleared");
        out(ECHO_CONSOLE, "Server bans cleared");
        out(ECHO_IRC, "All bans cleared");
    }

    void addban(uint ip, int expire)
    {
        allowedips.removeobj(ip);
        ban b;
        b.time = totalmillis;
        b.expire = totalmillis + expire;
        b.ip = ip;
        loopv(bannedips) if(b.expire < bannedips[i].expire) { bannedips.insert(i, b); return; }
        bannedips.add(b);
    }

    //QServ
    bool firstblood;
    bool enableautosendmap = true;
    bool q_teammode = false;
    bool persist = false;
    bool notgotitems = true; //true when map has changed and waiting for clients to send item
    bool gamepaused = false, shouldstep = true;
    int gamemillis = 0, gamelimit = 0, nextexceeded = 0;
    int gamespeed = 100, interm = 0;
    extern int gamemode = 0;
    string smapname = "";
    enet_uint32 lastsend = 0;
    int mastermode = MM_OPEN, mastermask = MM_PRIVSERV;
    stream *mapdata = NULL;

    vector<clientinfo *> connects, clients, bots;

    void kickclients(uint ip, clientinfo *actor = NULL)
    {
        loopvrev(clients)
        {
            clientinfo &c = *clients[i];
            if(c.state.aitype != AI_NONE || c.privilege >= PRIV_ADMIN || c.local) continue;
            if(actor && (c.privilege > actor->privilege || c.clientnum == actor->clientnum)) continue;
            if(getclientip(c.clientnum) == ip) disconnect_client(c.clientnum, DISC_KICK);
        }
    }

    VAR(serverflagruns, 0, 0, 1); //enable/disable flagrun message/storage
    struct _flagrun
    {
        char *map;
        int gamemode;
        char *name;
        int timeused;
    };
    vector<_flagrun> _flagruns;
    int _newflagrun = 0;
    void _doflagrun(clientinfo *ci, int timeused)
    {
        if(timeused <= 500)
        {
            out(ECHO_ALL, "\f0%s \f7scored an assisted flagrun",colorname(ci));
            return;
        }
        if(serverflagruns)
        {
            _flagrun *fr = 0;
            loopv(_flagruns) if(_flagruns[i].gamemode == gamemode && !strcmp(_flagruns[i].map, smapname))
            { fr = &_flagruns[i]; break; }
            bool isbest = false;
            if(!fr)
            {
                isbest = true;
                int lastfr = _flagruns.length();
                if(lastfr >= 1024) return;
                _flagruns.add();
                _flagruns[lastfr].map = newstring(smapname);
                _flagruns[lastfr].gamemode = gamemode;
                _flagruns[lastfr].name = newstring(ci->name);
                _flagruns[lastfr].timeused = timeused;
                fr = &_flagruns[lastfr];
            }
            isbest = isbest || timeused <= fr->timeused;
            if(isbest)
            {
                _newflagrun = 1;
                if(strcmp(ci->name, fr->name))
                {
                    DELETEA(fr->name);
                    fr->name = newstring(ci->name);
                }
                fr->timeused = timeused;
            }
            string msg;
            if(isbest) formatstring(msg)("\f0%s \f7scored a flagrun in \f7%i.%02i \f7seconds (\f2best\f7)",
                                         colorname(ci), timeused/1000, (timeused%1000)/10);
            else formatstring(msg)("\f0%s \f7scored a flagrun in \f7%i.%02i \f7seconds (\f2best: \f0%s \f7%i.%02i\f7)",
                                   colorname(ci), timeused/1000, (timeused%1000)/10, fr->name, fr->timeused/1000, (fr->timeused%1000)/10);
            sendservmsg(msg);
        }
    }

    void addflagrun(int mode, const char *map, int timeused, const char *name)
    {
        _flagrun *fr = 0;
        loopv(_flagruns) if(_flagruns[i].gamemode == mode && !strcmp(_flagruns[i].map, map))
        {
            fr = &_flagruns[i];
            break;
        }
        if(!fr)
        {
            int lastfr = _flagruns.length();
            if(lastfr >= 1024) return;
            _flagruns.add();
            _flagruns[lastfr].map = newstring(map);
            _flagruns[lastfr].gamemode = mode;
            _flagruns[lastfr].name = newstring(name);
            _flagruns[lastfr].timeused = timeused;
            fr = &_flagruns[lastfr];
        }

        if(strcmp(name, fr->name))
        {
            DELETEA(fr->name);
            fr->name = newstring(name);
        }
        fr->timeused = timeused;
    }
    ICOMMAND(flagrun, "isis", (int *i, const char *s, int *j, const char *z), addflagrun(*i, s, *j, z));

    void _storeflagruns()
    {
        if(serverflagruns)
        {
            stream *f = openutf8file(path("./config/flagruns.cfg", true), "w");
            if(f)
            {
                f->printf("//Automatically generated by QServ at exit: lists best flagruns\n\n");
                loopv(_flagruns)
                f->printf("flagrun %i \"%s\" %i \"%s\"\n", _flagruns[i].gamemode, _flagruns[i].map, _flagruns[i].timeused, _flagruns[i].name);
                delete f;
            }
        }
    }

    struct maprotation
    {
        static int exclude;
        long modes;
        string map;

        long calcmodemask() const { return modes&((long)1<<NUMGAMEMODES) ? modes & ~exclude : modes; }
        bool hasmode(int mode, int offset = STARTGAMEMODE) const { return (calcmodemask() & (1 << (mode-offset))) != 0; }

        int findmode(int mode) const
        {
            if(!hasmode(mode)) loopi(NUMGAMEMODES) if(hasmode(i, 0)) return i+STARTGAMEMODE;
            return mode;
        }

        bool match(int reqmode, const char *reqmap) const
        {
            return hasmode(reqmode) && (!map[0] || !reqmap[0] || !strcmp(map, reqmap));
        }

        bool includes(const maprotation &rot) const
        {
            return rot.modes == modes ? rot.map[0] && !map[0] : (rot.modes & modes) == rot.modes;
        }
    };
    int maprotation::exclude = 0;
    vector<maprotation> maprotations;
    int curmaprotation = 0;

    VAR(lockmaprotation, 0, 0, 2);

    void maprotationreset()
    {
        maprotations.setsize(0);
        curmaprotation = 0;
        maprotation::exclude = 0;
    }

    void nextmaprotation()
    {
        curmaprotation++;
        if(maprotations.inrange(curmaprotation) && maprotations[curmaprotation].modes) return;
        do curmaprotation--;
        while(maprotations.inrange(curmaprotation) && maprotations[curmaprotation].modes);
        curmaprotation++;
    }

    int findmaprotation(int mode, const char *map)
    {
        for(int i = max(curmaprotation, 0); i < maprotations.length(); i++)
        {
            maprotation &rot = maprotations[i];
            if(!rot.modes) break;
            if(rot.match(mode, map)) return i;
        }
        int start;
        for(start = max(curmaprotation, 0) - 1; start >= 0; start--) if(!maprotations[start].modes) break;
        start++;
        for(int i = start; i < curmaprotation; i++)
        {
            maprotation &rot = maprotations[i];
            if(!rot.modes) break;
            if(rot.match(mode, map)) return i;
        }
        int best = -1;
        loopv(maprotations)
        {
            maprotation &rot = maprotations[i];
            if(rot.match(mode, map) && (best < 0 || maprotations[best].includes(rot))) best = i;
        }
        return best;
    }


    bool searchmodename(const char *haystack, const char *needle)
    {
        if(!needle[0]) return true;
        do
        {
            if(needle[0] != '.')
            {
                haystack = strchr(haystack, needle[0]);
                if(!haystack) break;
                haystack++;
            }
            const char *h = haystack, *n = needle+1;
            for(; *h && *n; h++)
            {
                if(*h == *n) n++;
                else if(*h != ' ') break;
            }
            if(!*n) return true;
            if(*n == '.') return !*h;
        } while(needle[0] != '.');
        return false;
    }

    int genmodemask(vector<char *> &modes)
    {
        long modemask = 0;
        loopv(modes)
        {
            const char *mode = modes[i];
            int op = mode[0];
            switch(mode[0])
            {
                case '*':
                    modemask |= (long)1<<NUMGAMEMODES;
                    loopk(NUMGAMEMODES) if(m_checknot(k+STARTGAMEMODE, M_DEMO|M_EDIT|M_LOCAL)) modemask |= 1<<k;
                    continue;
                case '!':
                    mode++;
                    if(mode[0] != '?') break;
                case '?':
                    mode++;
                    loopk(NUMGAMEMODES) if(searchmodename(gamemodes[k].name, mode))
                    {
                        if(op == '!') modemask &= ~(1<<k);
                        else modemask |= 1<<k;
                    }
                    continue;
            }
            int modenum = INT_MAX;
            if(isdigit(mode[0])) modenum = atoi(mode);
            else loopk(NUMGAMEMODES) if(searchmodename(gamemodes[k].name, mode)) { modenum = k+STARTGAMEMODE; break; }
            if(!m_valid(modenum)) continue;
            switch(op)
            {
                case '!': modemask &= ~(1 << (modenum - STARTGAMEMODE)); break;
                default: modemask |= 1 << (modenum - STARTGAMEMODE); break;
            }
        }
        return modemask;
    }

    bool addmaprotation(int modemask, const char *map)
    {
        if(!map[0]) loopk(NUMGAMEMODES) if(modemask&(1<<k) && !m_check(k+STARTGAMEMODE, M_EDIT)) modemask &= ~(1<<k);
        if(!modemask) return false;
        if(!(modemask&((long)1<<NUMGAMEMODES))) maprotation::exclude |= modemask;
        maprotation &rot = maprotations.add();
        rot.modes = modemask;
        copystring(rot.map, map);
        return true;
    }

    void addmaprotations(tagval *args, int numargs)
    {
        vector<char *> modes, maps;
        for(int i = 0; i + 1 < numargs; i += 2)
        {
            explodelist(args[i].getstr(), modes);
            explodelist(args[i+1].getstr(), maps);
            int modemask = genmodemask(modes);
            if(maps.length()) loopvj(maps) addmaprotation(modemask, maps[j]);
            else addmaprotation(modemask, "");
            modes.deletearrays();
            maps.deletearrays();
        }
        if(maprotations.length() && maprotations.last().modes)
        {
            maprotation &rot = maprotations.add();
            rot.modes = 0;
            rot.map[0] = '\0';
        }
    }

    COMMAND(maprotationreset, "");
    COMMANDN(maprotation, addmaprotations, "ss2V");

    struct demofile
    {
        string info;
        uchar *data;
        int len;
    };

    vector<demofile> demos;
    bool demonextmatch = false;
    stream *demotmp = NULL, *demorecord = NULL, *demoplayback = NULL;
    int nextplayback = 0, demomillis = 0;

    /***********QServ***********/
    SVAR(serverdesc, "");
    SVAR(serverpass, "");
    SVAR(adminpass, "");
    SVAR(servermotd, "");
    SVAR(sweartext, "");
    SVAR(spreesuicidemsg, "");
    SVAR(spreefinmsg, "");
    SVAR(defmultikillmsg, "MULTI KILL");
    SVAR(pingwarncustommsg, "");
    SVAR(defaultmap, "");
    SVAR(defaultmodename, "");

    int64_t lastfragmillis;
    int multifrags;
    int spreefrags;

    VAR(minspreefrags, 2, 5, INT_MAX);               //minimum number of kills for a killing spree to occur
    VAR(multifragmillis, 1, 2000, INT_MAX);          //milliseconds between multi-kill messages
    VAR(maxpingwarn, 1, 1000, INT_MAX);              //maximum ping before a client is warned about their ping
    VAR(minmultikill, 2, 2, INT_MAX);                //minimum number of kills for a multi-kill to occur
    VAR(spammillis, 1, 1000, INT_MAX);               //interval for spam detection
    VAR(maxspam, 2, 3, INT_MAX);                     //number of lines that you can type in spammillis interval without getting blocked
    VAR(maxteamkills, 1, 100, INT_MAX);              //max teamkill number for message
    VAR(clearbansonempty, 0, 1, 1);                  //enables/disables clearing bans when the server empties of players
    VAR(maxdemos, 0, 5, 25);                         //maximum demos stored on the server
    VAR(maxdemosize, 0, 16, 64);                     //sets the max demo size for packets per demo
    VAR(restrictdemos, 0, 1, 1);                     //restircts recording demos for masters/admins/nopriv
    VAR(restrictpausegame, 0, 1, 1);                 //restricts setting pausegame for masters/admin/nopriv
    VAR(restrictgamespeed, 0, 1, 1);                 //restricts setting gamespeed for masters/admin/nopriv
    VAR(autodemo, 0, 1, 1);                          //record demos automatically
    VAR(welcomewithname, 0, 1, 1);                   //welcome a client with name
    VAR(serverconnectmsg, 0, 1, 1);                  //incoming connection alerts for admins
    VAR(nodamage, 0, 0, 1);                          //no damage for anyone
    VAR(notkdamage, 0, 0, 1);                        //no damage for teamkills
    VAR(autosendmap, 0, 1, 1);                       //automatically sends map in edit mode
    VAR(instacoop, 0, 0, 1);                         //insta like characteristics of edit mode
    VAR(instacoop_gamelimit, 1000, 600000, 9999999); //time limit for instacoop games
    VAR(enable_passflag, 0, 1, 1);                   //enables pass the flag in ctf modes
    VAR(no_single_private, 0, 0, 1);                 //no single user can set mastermode private (requires at least 2 clients/admins are exempt)
    VAR(enablemultiplemasters, 0, 0, 1);             //enables /setmaster 1 for multiple clients (stops need for #sendprivs or givemaster)

    VARF(publicserver, 0, 0, 2, {
        switch(publicserver)
        {
            case 0: default: mastermask = MM_PRIVSERV; break;
            case 1: mastermask = MM_PUBSERV; break;
            case 2: mastermask = MM_COOPSERV; break;
        }
    });

    static const struct { const char *name; int timediv; } timedivinfos[] =
    {
        // month is inaccurate
        { "week", 60*60*24*7 },
        { "day", 60*60*24 },
        { "hour", 60*60 },
        { "minute", 60 },
        { "second", 1 }
    };

    void formatsecs(vector<char> &timebuf, uint secs)
    {
        bool moded = false;
        const size_t tl = sizeof(timedivinfos)/sizeof(timedivinfos[0]);
        for(size_t i = 0; i < tl; i++)
        {
            uint t = secs / timedivinfos[i].timediv;
            if(!t && (i+1<tl || moded)) continue;
            secs %= timedivinfos[i].timediv;
            if(moded) timebuf.add(' ');
            moded = true;
            charbuf b = timebuf.reserve(10 + 1);
            int blen = b.remaining();
            int plen = snprintf(b.buf, blen, "%u", t);
            timebuf.advance(clamp(plen, 0, blen-1));
            timebuf.add(' ');
            timebuf.put(timedivinfos[i].name, strlen(timedivinfos[i].name));
            if(t != 1) timebuf.add('s');
            if(!secs) break;
        }
    }

    uint mspassed;
    void send_connected_time(clientinfo *ci, int sender) {
        mspassed = uint(totalmillis-ci->connectmillis);
        if(mspassed/1000 != 0)
        {
            vector<char> timebuf;
            formatsecs(timebuf, mspassed/1000);
            if(timebuf.length())
            {
                timebuf.add(0);
                sendf(sender, 1, "ris", N_SERVMSG, tempformatstring("connected: %s ago", timebuf.getbuf()));
            }
        }
    }

    //Enable or disable master
    void switchallowmaster() {
        mastermask = MM_PRIVSERV;
    }
    void switchdisallowmaster() {
        mastermask = MM_PUBSERV;
    }

    struct teamkillkick
    {
        int modes, limit, ban;

        bool match(int mode) const
        {
            return (modes&(1<<(mode-STARTGAMEMODE)))!=0;
        }

        bool includes(const teamkillkick &tk) const
        {
            return tk.modes != modes && (tk.modes & modes) == tk.modes;
        }
    };
    vector<teamkillkick> teamkillkicks;

    void teamkillkickreset()
    {
        teamkillkicks.setsize(0);
    }

    void addteamkillkick(char *modestr, int *limit, int *ban)
    {
        vector<char *> modes;
        explodelist(modestr, modes);
        teamkillkick &kick = teamkillkicks.add();
        kick.modes = genmodemask(modes);
        kick.limit = *limit;
        kick.ban = *ban > 0 ? *ban*60000 : (*ban < 0 ? 0 : 30*60000);
        modes.deletearrays();
    }

    COMMAND(teamkillkickreset, "");
    COMMANDN(teamkillkick, addteamkillkick, "sii");

    struct teamkillinfo
    {
        uint ip;
        int teamkills;
    };
    vector<teamkillinfo> teamkills;
    bool shouldcheckteamkills = false;

    void addteamkill(clientinfo *actor, clientinfo *victim, int n)
    {
        if(!m_timed || actor->state.aitype != AI_NONE || actor->local || actor->privilege || (victim && victim->state.aitype != AI_NONE)) return;
        shouldcheckteamkills = true;
        uint ip = getclientip(actor->clientnum);
        loopv(teamkills) if(teamkills[i].ip == ip)
        {
            teamkills[i].teamkills += n;
            return;
        }
        teamkillinfo &tk = teamkills.add();
        tk.ip = ip;
        tk.teamkills = n;
    }


    void checkteamkills()
    {
        teamkillkick *kick = NULL;
        if(m_timed) loopv(teamkillkicks) if(teamkillkicks[i].match(gamemode) && (!kick || kick->includes(teamkillkicks[i])))
            kick = &teamkillkicks[i];
        if(kick) loopvrev(teamkills)
        {
            teamkillinfo &tk = teamkills[i];
            if(tk.teamkills >= kick->limit)
            {
                if(kick->ban > 0) addban(tk.ip, kick->ban);
                kickclients(tk.ip);
                teamkills.removeunordered(i);
            }
        }
        shouldcheckteamkills = false;
    }

    void *newclientinfo() { return new clientinfo; }
    void deleteclientinfo(void *ci) { delete (clientinfo *)ci; }

    clientinfo *getinfo(int n)
    {
        if(n < MAXCLIENTS) return (clientinfo *)getclientinfo(n);
        n -= MAXCLIENTS;
        return bots.inrange(n) ? bots[n] : NULL;
    }

    uint mcrc = 0;
    vector<entity> ments;
    vector<server_entity> sents;
    vector<savedscore> scores;

    int msgsizelookup(int msg)
    {
        static int sizetable[NUMMSG] = { -1 };
        if(sizetable[0] < 0)
        {
            memset(sizetable, -1, sizeof(sizetable));
            for(const int *p = msgsizes; *p >= 0; p += 2) sizetable[p[0]] = p[1];
        }
        return msg >= 0 && msg < NUMMSG ? sizetable[msg] : -1;
    }

    const char *modename(int n, const char *unknown)
    {
        if(m_valid(n)) return gamemodes[n - STARTGAMEMODE].name;
        return unknown;
    }

    const char *mastermodename(int n, const char *unknown)
    {
        return (n>=MM_START && size_t(n-MM_START)<sizeof(mastermodenames)/sizeof(mastermodenames[0])) ? mastermodenames[n-MM_START] : unknown;
    }

    const char *privname(int type)
    {
        switch(type)
        {
            case PRIV_ADMIN: return "\f6admin";
            case PRIV_AUTH: return "\f8auth";
            case PRIV_MASTER: return "\f0master";
            default: return "\f1unknown";
        }
    }
    void sendservmsg(const char *s) { sendf(-1, 1, "ris", N_SERVMSG, s); }
    void sendservmsgf(const char *fmt, ...)
    {
        defvformatstring(s, fmt, fmt);
        sendf(-1, 1, "ris", N_SERVMSG, s);
    }

    // weapon accuracy
    int getwepaccuracy(int cn, int gun)
    {
        int acc = 0;
        clientinfo *ci = getinfo(cn);
        if(ci && gun>0 && gun<=NUMGUNS)
            acc = ci->state.guninfo[gun].damage*100/max(ci->state.guninfo[gun].shotdamage, 1);
        return(acc);
    }


    void resetitems()
    {
        mcrc = 0;
        ments.setsize(0);
        sents.setsize(0);
        //cps.reset();

    }

    bool serveroption(const char *arg)
    {
        if(arg[0]=='-') switch(arg[1])
        {
            case 'n': setsvar("serverdesc", &arg[2]); return true;
            case 'y': setsvar("serverpass", &arg[2]); return true;
            case 'p': setsvar("adminpass", &arg[2]); return true;
            case 'o': setvar("publicserver", atoi(&arg[2])); return true;
        }
        return false;
    }

    string blkmsg[3] = {"fuck", "shit", "cunt"};
    char textcmd(const char *a, const char *text){
        for (int b=0; b<strlen(a); b++) {
            if (text[b+1] != a[b]) {
                return false;
            }
        }
        if(text[strlen(a)+1] != ' ' && text[strlen(a)+1] != '\0') {
            for (int l=0; l<3; l++) {
                if (strcmp(blkmsg[l], a)) {
                    return true;
                    break;
                }
            }
            return false;
        }
        return true;
    }
    void textblk(const char *b, char *text, clientinfo *ci){
        bool bad=false;
        for (int a=0; a<strlen(text); a++) {
            if(textcmd(b, text+a-1)) bad=true;
        }
        if(bad){
            int n = randomMT() % 7 + 0; //replaced rand with threadsafe randomMT
            defformatstring(d)("\f%i%s \f0%s!", n, sweartext, ci->name);
            sendservmsg(d);
            bad=false;
        }
    }

    extern void changemap(const char *s, int mode);
    //Server initalizer
    void serverinit()
    {
        smapname[0] = '\0';
        resetitems();
        if(serverflagruns) execfile("./config/flagruns.cfg", false);
        int mc = 22; int gm;
        for(int i = 0; i <= mc; i++) {
            if(!strcmp(defaultmodename, qserv_modenames[i]))  {
                gm = i;
                changemap(defaultmap, gm);
                break;
            }
        }
    }
    //Server deinitalizer
    void serverclose()
    {
        _storeflagruns();
    }

    int numclients(int exclude = -1, bool nospec = true, bool noai = true, bool priv = false)
    {
        int n = 0;
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->clientnum!=exclude && (!nospec || ci->state.state!=CS_SPECTATOR || (priv && (ci->privilege || ci->local))) && (!noai || ci->state.aitype == AI_NONE)) n++;
        }
        return n;
    }

    struct servmode
    {
        virtual ~servmode() {}

        virtual void entergame(clientinfo *ci) {}
        virtual void leavegame(clientinfo *ci, bool disconnecting = false) {}

        virtual void moved(clientinfo *ci, const vec &oldpos, bool oldclip, const vec &newpos, bool newclip) {}
        virtual bool canspawn(clientinfo *ci, bool connecting = false) { return true; }
        virtual void spawned(clientinfo *ci) {}
        virtual int fragvalue(clientinfo *victim, clientinfo *actor)
        {
            if(victim==actor || isteam(victim->team, actor->team)) return -1;
            return 1;
        }
        virtual void died(clientinfo *victim, clientinfo *actor) {}
        virtual bool canchangeteam(clientinfo *ci, const char *oldteam, const char *newteam) { return true; }
        virtual void changeteam(clientinfo *ci, const char *oldteam, const char *newteam) {}
        virtual void initclient(clientinfo *ci, packetbuf &p, bool connecting) {}
        virtual void update() {}
        virtual void cleanup() {}
        virtual void setup() {}
        virtual void newmap() {}
        virtual void intermission() {}
        virtual bool hidefrags() { return false; }
        virtual int getteamscore(const char *team) { return 0; }
        virtual void getteamscores(vector<teamscore> &scores) {}
        virtual bool extinfoteam(const char *team, ucharbuf &p) { return false; }
    };

#define SERVMODE 1
#include "capture.h"
#include "ctf.h"
#include "collect.h"

    captureservmode capturemode;
    ctfservmode ctfmode;
    collectservmode collectmode;
    servmode *smode = NULL;

    bool canspawnitem(int type) { return !m_noitems && (type>=I_SHELLS && type<=I_QUAD && (!m_noammo || type<I_SHELLS || type>I_CARTRIDGES)); }

    int spawntime(int type)
    {
        if(m_classicsp) return INT_MAX;
        int np = numclients(-1, true, false);
        np = np<3 ? 4 : (np>4 ? 2 : 3); //spawn times dependent on numclients
        int sec = 0;
        switch(type)
        {
            case I_SHELLS:
            case I_BULLETS:
            case I_ROCKETS:
            case I_ROUNDS:
            case I_GRENADES:
            case I_CARTRIDGES: sec = np*4; break;
            case I_HEALTH: sec = np*5; break;
            case I_GREENARMOUR: sec = 20; break;
            case I_YELLOWARMOUR: sec = 30; break;
            case I_BOOST: sec = 60; break;
            case I_QUAD: sec = 70; break;
        }
        return sec*1000;
    }

    bool delayspawn(int type)
    {
        switch(type)
        {
            case I_GREENARMOUR:
            case I_YELLOWARMOUR:
                return !m_classicsp;
            case I_BOOST:
            case I_QUAD:
                return true;
            default:
                return false;
        }
    }

    bool pickup(int i, int sender) //server-side item pickup, acknowledge first client that gets it
    {
        if((m_timed && gamemillis>=gamelimit) || !sents.inrange(i) || !sents[i].spawned) return false;
        clientinfo *ci = getinfo(sender);
        if(!ci || (!ci->local && !ci->state.canpickup(sents[i].type))) return false;
        sents[i].spawned = false;
        sents[i].spawntime = spawntime(sents[i].type);
        sendf(-1, 1, "ri3", N_ITEMACC, i, sender);
        ci->state.pickup(sents[i].type);
        return true;
    }

    static hashset<teaminfo> teaminfos;

    void clearteaminfo()
    {
        teaminfos.clear();
    }

    bool teamhasplayers(const char *team) { loopv(clients) if(!strcmp(clients[i]->team, team)) return true; return false; }

    bool pruneteaminfo()
    {
        int oldteams = teaminfos.numelems;
        enumerates(teaminfos, teaminfo, old,
                   if(!old.frags && !teamhasplayers(old.team)) teaminfos.remove(old.team);
                   );
        return teaminfos.numelems < oldteams;
    }

    teaminfo *addteaminfo(const char *team)
    {
        teaminfo *t = teaminfos.access(team);
        if(!t)
        {
            if(teaminfos.numelems >= MAXTEAMS && !pruneteaminfo()) return NULL;
            t = &teaminfos[team];
            copystring(t->team, team, sizeof(t->team));
            t->frags = 0;
        }
        return t;
    }

    clientinfo *choosebestclient(float &bestrank)
    {
        clientinfo *best = NULL;
        bestrank = -1;
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->state.timeplayed<0) continue;
            float rank = ci->state.state!=CS_SPECTATOR ? ci->state.effectiveness/max(ci->state.timeplayed, 1) : -1;
            if(!best || rank > bestrank) { best = ci; bestrank = rank; }
        }
        return best;
    }

    void autoteam()
    {
        static const char * const teamnames[2] = {"good", "evil"};
        vector<clientinfo *> team[2];
        float teamrank[2] = {0, 0};
        for(int round = 0, remaining = clients.length(); remaining>=0; round++)
        {
            int first = round&1, second = (round+1)&1, selected = 0;
            while(teamrank[first] <= teamrank[second])
            {
                float rank;
                clientinfo *ci = choosebestclient(rank);
                if(!ci) break;
                if(smode && smode->hidefrags()) rank = 1;
                else if(selected && rank<=0) break;
                ci->state.timeplayed = -1;
                team[first].add(ci);
                if(rank>0) teamrank[first] += rank;
                selected++;
                if(rank<=0) break;
            }
            if(!selected) break;
            remaining -= selected;
        }
        loopi(sizeof(team)/sizeof(team[0]))
        {
            addteaminfo(teamnames[i]);
            loopvj(team[i])
            {
                clientinfo *ci = team[i][j];
                if(!strcmp(ci->team, teamnames[i])) continue;
                copystring(ci->team, teamnames[i], MAXTEAMLEN+1);
                sendf(-1, 1, "riisi", N_SETTEAM, ci->clientnum, teamnames[i], -1);
            }
        }
    }

    struct teamrank
    {
        const char *name;
        float rank;
        int clients;

        teamrank(const char *name) : name(name), rank(0), clients(0) {}
    };

    const char *chooseworstteam(const char *suggest = NULL, clientinfo *exclude = NULL)
    {
        teamrank teamranks[2] = {teamrank("good"),teamrank("evil")};
        const int numteams = sizeof(teamranks)/sizeof(teamranks[0]);
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci==exclude || ci->state.aitype!=AI_NONE || ci->state.state==CS_SPECTATOR || !ci->team[0]) continue;
            ci->state.timeplayed += lastmillis - ci->state.lasttimeplayed;
            ci->state.lasttimeplayed = lastmillis;

            loopj(numteams) if(!strcmp(ci->team, teamranks[j].name))
            {
                teamrank &ts = teamranks[j];
                ts.rank += ci->state.effectiveness/max(ci->state.timeplayed, 1);
                ts.clients++;
                break;
            }
        }
        teamrank *worst = &teamranks[numteams-1];
        loopi(numteams-1)
        {
            teamrank &ts = teamranks[i];
            if(smode && smode->hidefrags())
            {
                if(ts.clients < worst->clients || (ts.clients == worst->clients && ts.rank < worst->rank)) worst = &ts;
            }
            else if(ts.rank < worst->rank || (ts.rank == worst->rank && ts.clients < worst->clients)) worst = &ts;
        }
        return worst->name;
    }

    void prunedemos(int extra = 0)
    {
        int n = clamp(demos.length() + extra - maxdemos, 0, demos.length());
        if(n <= 0) return;
        loopi(n) delete[] demos[n].data;
        demos.remove(0, n);
    }

    void adddemo()
    {
        if(!demotmp) return;
        int len = (int)min(demotmp->size(), stream::offset((maxdemosize<<20) + 0x10000));
        demofile &d = demos.add();
        time_t t = time(NULL);
        char *timestr = ctime(&t), *trim = timestr + strlen(timestr);
        while(trim>timestr && iscubespace(*--trim)) *trim = '\0';
        formatstring(d.info)("%s: %s, %s, %.2f%s", timestr, modename(gamemode), smapname, len > 1024*1024 ? len/(1024*1024.f) : len/1024.0f, len > 1024*1024 ? "MB" : "kB");
        sendservmsgf("\f7Demo \"%s\" recorded", d.info);
        d.data = new uchar[len];
        d.len = len;
        demotmp->seek(0, SEEK_SET);
        demotmp->read(d.data, len);
        DELETEP(demotmp);
    }

    void enddemorecord()
    {
        if(!demorecord) return;

        DELETEP(demorecord);

        if(!demotmp) return;
        if(!maxdemos || !maxdemosize) { DELETEP(demotmp); return; }

        prunedemos(1);
        adddemo();
    }

    void writedemo(int chan, void *data, int len)
    {
        if(!demorecord) return;
        int stamp[3] = { gamemillis, chan, len };
        lilswap(stamp, 3);
        demorecord->write(stamp, sizeof(stamp));
        demorecord->write(data, len);
        if(demorecord->rawtell() >= (maxdemosize<<20)) enddemorecord();
    }

    void recordpacket(int chan, void *data, int len)
    {
        return;
    }

    int welcomepacket(packetbuf &p, clientinfo *ci);
    void sendwelcome(clientinfo *ci);

    void setupdemorecord()
    {
        if(!m_mp(gamemode) || m_edit) return;

        demotmp = opentempfile("demorecord", "w+b");
        if(!demotmp) return;

        stream *f = opengzfile(NULL, "wb", demotmp);
        if(!f) { DELETEP(demotmp); return; }

        sendservmsg("\f7Recording demo...");

        demorecord = f;

        demoheader hdr;
        memcpy(hdr.magic, DEMO_MAGIC, sizeof(hdr.magic));
        hdr.version = DEMO_VERSION;
        hdr.protocol = PROTOCOL_VERSION;
        lilswap(&hdr.version, 2);
        demorecord->write(&hdr, sizeof(demoheader));

        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
        welcomepacket(p, NULL);
        writedemo(1, p.buf, p.len);
    }

    void listdemos(int cn)
    {
        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
        putint(p, N_SENDDEMOLIST);
        putint(p, demos.length());
        loopv(demos) sendstring(demos[i].info, p);
        sendpacket(cn, 1, p.finalize());
    }

    void cleardemos(int n)
    {
        if(!n)
        {
            loopv(demos) delete[] demos[i].data;
            demos.shrink(0);
            sendservmsg("\f7Deleted all demos");
        }
        else if(demos.inrange(n-1))
        {
            delete[] demos[n-1].data;
            demos.remove(n-1);
            sendservmsgf("\f7Deleted demo: \f2%d", n);
        }
    }

    static void freegetmap(ENetPacket *packet)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->getmap == packet) ci->getmap = NULL;
        }
    }

    static void freegetdemo(ENetPacket *packet)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->getdemo == packet) ci->getdemo = NULL;
        }
    }

    void senddemo(clientinfo *ci, int num, int tag)
    {
        if(ci->getdemo) return;
        if(!num) num = demos.length();
        if(!demos.inrange(num-1)) return;
        demofile &d = demos[num-1];
        if((ci->getdemo = sendf(ci->clientnum, 2, "riim", N_SENDDEMO, tag, d.len, d.data)))
            ci->getdemo->freeCallback = freegetdemo;
    }

    void enddemoplayback()
    {
        if(!demoplayback) return;
        DELETEP(demoplayback);

        loopv(clients) sendf(clients[i]->clientnum, 1, "ri3", N_DEMOPLAYBACK, 0, clients[i]->clientnum);

        out(ECHO_ALL,"Demo finished playing");

        loopv(clients) sendwelcome(clients[i]);
    }

    void setupdemoplayback()
    {
        if(demoplayback) return;
        demoheader hdr;
        string msg;
        msg[0] = '\0';
        defformatstring(file)("%s.dmo", smapname);
        demoplayback = opengzfile(file, "rb");
        if(!demoplayback) formatstring(msg)("\f3Error: couldn't read demo \"%s\"", file);
        else if(demoplayback->read(&hdr, sizeof(demoheader))!=sizeof(demoheader) || memcmp(hdr.magic, DEMO_MAGIC, sizeof(hdr.magic)))
            formatstring(msg)("\f3Error: \"%s\" is not a demo file", file);
        else
        {
            lilswap(&hdr.version, 2);
            if(hdr.version!=DEMO_VERSION) formatstring(msg)("\f3Error: Demo \"%s\" requires an %s version of Cube 2: Sauerbraten", file, hdr.version<DEMO_VERSION ? "older" : "newer");
            else if(hdr.protocol!=PROTOCOL_VERSION) formatstring(msg)("\f3Error: Demo \"%s\" requires an %s version of Cube 2: Sauerbraten", file, hdr.protocol<PROTOCOL_VERSION ? "older" : "newer");
        }
        if(msg[0])
        {
            DELETEP(demoplayback);
            sendservmsg(msg);
            return;
        }

        sendservmsgf("\f7Playing demo \"%s\"", file);
        demomillis = 0;
        sendf(-1, 1, "ri3", N_DEMOPLAYBACK, 1, -1);

        if(demoplayback->read(&nextplayback, sizeof(nextplayback))!=sizeof(nextplayback))
        {
            enddemoplayback();
            return;
        }
        lilswap(&nextplayback, 1);
    }

    void readdemo()
    {
        if(!demoplayback) return;
        demomillis += curtime;
        while(demomillis>=nextplayback)
        {
            int chan, len;
            if(demoplayback->read(&chan, sizeof(chan))!=sizeof(chan) ||
               demoplayback->read(&len, sizeof(len))!=sizeof(len))
            {
                enddemoplayback();
                return;
            }
            lilswap(&chan, 1);
            lilswap(&len, 1);
            ENetPacket *packet = enet_packet_create(NULL, len+1, 0);
            if(!packet || demoplayback->read(packet->data+1, len)!=size_t(len)) //check size not just int
            {
                if(packet) enet_packet_destroy(packet);
                enddemoplayback();
                return;
            }
            packet->data[0] = N_DEMOPACKET;
            sendpacket(-1, chan, packet);
            if(!packet->referenceCount) enet_packet_destroy(packet);
            if(!demoplayback) break;
            if(demoplayback->read(&nextplayback, sizeof(nextplayback))!=sizeof(nextplayback))
            {
                enddemoplayback();
                return;
            }
            lilswap(&nextplayback, 1);
        }
    }

    void stopdemo()
    {
        if(m_demo) enddemoplayback();
        else enddemorecord();
    }

    void pausegame(bool val, clientinfo *ci = NULL)
    {
        if(gamepaused==val) return;
        gamepaused = val;
        sendf(-1, 1, "riii", N_PAUSEGAME, gamepaused ? 1 : 0, ci ? ci->clientnum : -1);
    }
    ICOMMAND(pausegame, "b", (int *val), pausegame(*val != 0));

    void checkpausegame()
    {
        if(!gamepaused) return;
        int admins = 0;
        loopv(clients) if(clients[i]->privilege >= (restrictpausegame ? PRIV_ADMIN : PRIV_MASTER) || clients[i]->local) admins++;
        if(!admins) pausegame(false);
    }

    void forcepaused(bool paused)
    {
        pausegame(paused);
    }

    bool ispaused() { return gamepaused; }

    void changegamespeed(int val, clientinfo *ci = NULL)
    {
        val = clamp(val, 10, 1000);
        if(val!=100 && m_ctf) loopv(clients) clients[i]->_xi.lasttakeflag = 0;
        if(gamespeed==val) return;
        gamespeed = val;
        sendf(-1, 1, "riii", N_GAMESPEED, gamespeed, ci ? ci->clientnum : -1);
    }

    void forcegamespeed(int speed)
    {
        changegamespeed(speed);
    }

    int scaletime(int t) { return t*gamespeed; }

    SVAR(serverauth, "");

    struct userkey
    {
        char *name;
        char *desc;

        userkey() : name(NULL), desc(NULL) {}
        userkey(char *name, char *desc) : name(name), desc(desc) {}
    };

    static inline uint hthash(const userkey &k) { return ::hthash(k.name); }
    static inline bool htcmp(const userkey &x, const userkey &y) { return !strcmp(x.name, y.name) && !strcmp(x.desc, y.desc); }

    struct userinfo : userkey
    {
        void *pubkey;
        int privilege;

        userinfo() : pubkey(NULL), privilege(PRIV_NONE) {}
        ~userinfo() { delete[] name; delete[] desc; if(pubkey) freepubkey(pubkey); }
    };
    hashset<userinfo> users;

    void adduser(char *name, char *desc, char *pubkey, char *priv)
    {
        userkey key(name, desc);
        userinfo &u = users[key];
        if(u.pubkey) { freepubkey(u.pubkey); u.pubkey = NULL; }
        if(!u.name) u.name = newstring(name);
        if(!u.desc) u.desc = newstring(desc);
        u.pubkey = parsepubkey(pubkey);
        switch(priv[0])
        {
            case 'a': case 'A': u.privilege = PRIV_ADMIN; break;
            case 'm': case 'M': default: u.privilege = PRIV_AUTH; break;
        }
    }
    COMMAND(adduser, "ssss");
    void clearusers()
    {
        users.clear();
    }
    COMMAND(clearusers, "");

    void hashpassword(int cn, int sessionid, const char *pwd, char *result, int maxlen)
    {
        char buf[2*sizeof(string)];
        formatstring(buf)("%d %d ", cn, sessionid);
        copystring(&buf[strlen(buf)], pwd);
        if(!hashstring(buf, result, maxlen)) *result = '\0';
    }

    bool checkpassword(clientinfo *ci, const char *wanted, const char *given)
    {
        string hash;
        hashpassword(ci->clientnum, ci->sessionid, wanted, hash, sizeof(hash));
        return !strcmp(hash, given);
    }

    void revokemaster(clientinfo *ci)
    {
        ci->privilege = PRIV_NONE;
        if(ci->state.state==CS_SPECTATOR && !ci->local) aiman::removeai(ci);
    }
    extern void connected(clientinfo *ci);

    bool setmaster(clientinfo *ci, bool val, const char *pass = "", const char *authname = NULL, const char *authdesc = NULL, int authpriv = PRIV_MASTER, bool force = false, bool trial = false, bool revoke = false)
    {
        if(authname && !val) return false;
        const char *name = "";
        if(val)
        {
            bool haspass = adminpass[0] && checkpassword(ci, adminpass, pass);
            int wantpriv = ci->local || haspass ? PRIV_ADMIN : authpriv;
            if(ci->privilege)
            {
                if(wantpriv <= ci->privilege) return true;
            }
            else if(wantpriv <= PRIV_MASTER && !force)
            {
                if(ci->state.state==CS_SPECTATOR)
                {
                    sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Spectators may not claim master");
                    return false;
                }
                if(!enablemultiplemasters) {
                    loopv(clients) if(ci!=clients[i] && clients[i]->privilege)
                    {
                        sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Someone already has privileges");
                        return false;
                    }
                }
                if(!authname && !(mastermask&MM_AUTOAPPROVE) && !ci->privilege && !ci->local)
                {
                    sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Master is disabled. \"/auth\" is enabled.");
                    return false;
                }
            }
            if(trial) return true;
            ci->privilege = wantpriv;
            name = privname(ci->privilege);
        }
        else
        {
            if(!ci->privilege) return false;
            if(trial) return true;
            name = privname(ci->privilege);
            revokemaster(ci);
        }
        bool hasmaster = false;
        loopv(clients) if(clients[i]->local || clients[i]->privilege >= PRIV_MASTER) hasmaster = true;
        if(!hasmaster)
        {
            mastermode = MM_OPEN;
            allowedips.shrink(0);
        }
        string msg;
        if(val && authname)
        {
            if(authdesc && authdesc[0]) formatstring(msg)("\f0%s \f7claimed \f6%s \f7as '\fs\f5%s\fr' [\fs\f0%s\fr]", colorname(ci), name, authname, authdesc);
            else formatstring(msg)("\f0%s \f7claimed %s as '\fs\f5%s\fr'", colorname(ci), name, authname);
        }
        else if(!revoke) formatstring(msg)("\f0%s \f7%s \f7%s", colorname(ci), val ? "claimed" : "relinquished", name);
        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);

        if(!revoke) {
            putint(p, N_SERVMSG);
            sendstring(msg, p);
        }

        putint(p, N_CURRENTMASTER);
        putint(p, mastermode);
        loopv(clients) if(clients[i]->privilege >= PRIV_MASTER && !clients[i]->isInvAdmin)
        {
            putint(p, clients[i]->clientnum);
            putint(p, clients[i]->privilege);
        }
        putint(p, -1);
        sendpacket(-1, 1, p.finalize());
        checkpausegame();
        return true;
    }

    void grantmaster(int cn)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if (ci->clientnum != cn) continue;
            setmaster(ci, true, "", NULL, NULL, PRIV_MASTER, true);
        }
    }
    ICOMMAND(grantmaster, "i", (int *val), grantmaster(*val));

    bool trykick(clientinfo *ci, int victim, const char *reason = NULL, const char *authname = NULL, const char *authdesc = NULL, int authpriv = PRIV_NONE, bool trial = false)
    {
        int priv = ci->privilege;
        if(authname)
        {
            if(priv >= authpriv || ci->local) authname = authdesc = NULL;
            else priv = authpriv;
        }
        if((priv || ci->local) && ci->clientnum!=victim)
        {
            clientinfo *vinfo = (clientinfo *)getclientinfo(victim);
            if(vinfo && (priv >= vinfo->privilege || ci->local) && vinfo->privilege < PRIV_ADMIN && !vinfo->local)
            {
                if(trial) return true;
                string kicker;
                if(authname)
                {
                    if(authdesc && authdesc[0]) formatstring(kicker)("%s as '\fs\f5%s\fr' [\fs\f0%s\fr]", colorname(ci), authname, authdesc);
                    else formatstring(kicker)("%s as '\fs\f5%s\fr'", colorname(ci), authname);
                }
                else copystring(kicker, colorname(ci));
                if(reason && reason[0]) sendservmsgf("\f0%s \f7kicked \f3%s \f7because: %s", kicker, colorname(vinfo), reason);
                else sendservmsgf("\f0%s \f7kicked \f3%s", kicker, colorname(vinfo));
                uint ip = getclientip(victim);
                addban(ip, 4*60*60000);
                kickclients(ip, ci);
            }
        }
        return false;
    }

    savedscore *findscore(clientinfo *ci, bool insert)
    {
        uint ip = getclientip(ci->clientnum);
        if(!ip && !ci->local) return 0;
        if(!insert)
        {
            loopv(clients)
            {
                clientinfo *oi = clients[i];
                if(oi->clientnum != ci->clientnum && getclientip(oi->clientnum) == ip && !strcmp(oi->name, ci->name))
                {
                    oi->state.timeplayed += lastmillis - oi->state.lasttimeplayed;
                    oi->state.lasttimeplayed = lastmillis;
                    static savedscore curscore;
                    curscore.save(oi->state);
                    return &curscore;
                }
            }
        }
        loopv(scores)
        {
            savedscore &sc = scores[i];
            if(sc.ip == ip && !strcmp(sc.name, ci->name)) return &sc;
        }
        if(!insert) return 0;
        savedscore &sc = scores.add();
        sc.ip = ip;
        copystring(sc.name, ci->name);
        return &sc;
    }

    void savescore(clientinfo *ci)
    {
        savedscore *sc = findscore(ci, true);
        if(sc) sc->save(ci->state);
    }

    int checktype(int type, clientinfo *ci)
    {
        if(ci)
        {
            if(!ci->connected) return type == (ci->connectauth ? N_AUTHANS : N_CONNECT) || type == N_PING ? type : -1;
            if(ci->local) return type;
        }
        // only allow edit messages in coop-edit mode
        if(type>=N_EDITENT && type<=N_EDITVAR && !m_edit) return -1;
        // server only messages
        static const int servtypes[] = { N_SERVINFO, N_INITCLIENT, N_WELCOME, N_MAPCHANGE, N_SERVMSG, N_DAMAGE, N_HITPUSH, N_SHOTFX, N_EXPLODEFX, N_DIED, N_SPAWNSTATE, N_FORCEDEATH, N_TEAMINFO, N_ITEMACC, N_ITEMSPAWN, N_TIMEUP, N_CDIS, N_CURRENTMASTER, N_PONG, N_RESUME, N_BASESCORE, N_BASEINFO, N_BASEREGEN, N_ANNOUNCE, N_SENDDEMOLIST, N_SENDDEMO, N_DEMOPLAYBACK, N_SENDMAP, N_DROPFLAG, N_SCOREFLAG, N_RETURNFLAG, N_RESETFLAG, N_INVISFLAG, N_CLIENT, N_AUTHCHAL, N_INITAI, N_EXPIRETOKENS, N_DROPTOKENS, N_STEALTOKENS, N_DEMOPACKET };
        if(ci)
        {
            loopi(sizeof(servtypes)/sizeof(int)) if(type == servtypes[i]) return -1;
            if(type < N_EDITENT || type > N_EDITVAR || !m_edit)
            {
                if(type != N_POS && ++ci->overflow >= 200) return -2;
            }
        }
        return type;
    }

    struct worldstate
    {
        int uses, len;
        uchar *data;

        worldstate() : uses(0), len(0), data(NULL) {}

        void setup(int n) { len = n; data = new uchar[n]; }
        void cleanup() { DELETEA(data); len = 0; }
        bool contains(const uchar *p) const { return p >= data && p < &data[len]; }
    };
    vector<worldstate> worldstates;
    bool reliablemessages = false;

    void cleanworldstate(ENetPacket *packet)
    {
        loopv(worldstates)
        {
            worldstate &ws = worldstates[i];
            if(!ws.contains(packet->data)) continue;
            ws.uses--;
            if(ws.uses <= 0)
            {
                ws.cleanup();
                worldstates.removeunordered(i);
            }
            break;
        }
    }

    void flushclientposition(clientinfo &ci)
    {
        if(ci.position.empty() || (!hasnonlocalclients() && !demorecord)) return;
        packetbuf p(ci.position.length(), 0);
        p.put(ci.position.getbuf(), ci.position.length());
        ci.position.setsize(0);
        sendpacket(-1, 0, p.finalize(), ci.ownernum);
    }

    static void sendpositions(worldstate &ws, ucharbuf &wsbuf)
    {
        if(wsbuf.empty()) return;
        int wslen = wsbuf.length();
        recordpacket(0, wsbuf.buf, wslen);
        wsbuf.put(wsbuf.buf, wslen);
        loopv(clients)
        {
            clientinfo &ci = *clients[i];
            if(ci.state.aitype != AI_NONE) continue;
            uchar *data = wsbuf.buf;
            int size = wslen;
            if(ci.wsdata >= wsbuf.buf) { data = ci.wsdata + ci.wslen; size -= ci.wslen; }
            if(size <= 0) continue;
            ENetPacket *packet = enet_packet_create(data, size, ENET_PACKET_FLAG_NO_ALLOCATE);
            sendpacket(ci.clientnum, 0, packet);
            if(packet->referenceCount) { ws.uses++; packet->freeCallback = cleanworldstate; }
            else enet_packet_destroy(packet);
        }
        wsbuf.offset(wsbuf.length());
    }

    static inline void addposition(worldstate &ws, ucharbuf &wsbuf, int mtu, clientinfo &bi, clientinfo &ci)
    {
        if(bi.position.empty()) return;
        if(wsbuf.length() + bi.position.length() > mtu) sendpositions(ws, wsbuf);
        int offset = wsbuf.length();
        wsbuf.put(bi.position.getbuf(), bi.position.length());
        bi.position.setsize(0);
        int len = wsbuf.length() - offset;
        if(ci.wsdata < wsbuf.buf) { ci.wsdata = &wsbuf.buf[offset]; ci.wslen = len; }
        else ci.wslen += len;
    }

    static void sendmessages(worldstate &ws, ucharbuf &wsbuf)
    {
        if(wsbuf.empty()) return;
        int wslen = wsbuf.length();
        recordpacket(1, wsbuf.buf, wslen);
        wsbuf.put(wsbuf.buf, wslen);

        loopv(clients)
        {
            clientinfo &ci = *clients[i];
            if(ci.state.aitype != AI_NONE) continue;
            uchar *data = wsbuf.buf;
            int size = wslen;
            if(ci.wsdata >= wsbuf.buf) { data = ci.wsdata + ci.wslen; size -= ci.wslen; }
            if(size <= 0) continue;
            ENetPacket *packet = enet_packet_create(data, size, (reliablemessages ? ENET_PACKET_FLAG_RELIABLE : 0) | ENET_PACKET_FLAG_NO_ALLOCATE);
            sendpacket(ci.clientnum, 1, packet);
            if(packet->referenceCount) { ws.uses++; packet->freeCallback = cleanworldstate; }
            else enet_packet_destroy(packet);
        }
        wsbuf.offset(wsbuf.length());
    }

    static void sendeditmessage(int sender, packetbuf *p, int offset)
    {
        ucharbuf wsbuf(p->buf+offset, p->maxlen);
        ENetPacket *packet = enet_packet_create(wsbuf.buf, wsbuf.maxlen, (reliablemessages ? ENET_PACKET_FLAG_RELIABLE : 0) | ENET_PACKET_FLAG_NO_ALLOCATE);
        sendedit(sender, packet);
        enet_packet_destroy(packet);
    }

    static inline void addmessages(worldstate &ws, ucharbuf &wsbuf, int mtu, clientinfo &bi, clientinfo &ci)
    {
        if(bi.messages.empty()) return;
        if(wsbuf.length() + 10 + bi.messages.length() > mtu) sendmessages(ws, wsbuf);
        int offset = wsbuf.length();
        putint(wsbuf, N_CLIENT);
        putint(wsbuf, bi.clientnum);
        putuint(wsbuf, bi.messages.length());
        wsbuf.put(bi.messages.getbuf(), bi.messages.length());
        bi.messages.setsize(0);
        int len = wsbuf.length() - offset;
        if(ci.wsdata < wsbuf.buf) { ci.wsdata = &wsbuf.buf[offset]; ci.wslen = len; }
        else ci.wslen += len;
    }

    bool buildworldstate()
    {
        int wsmax = 0;
        loopv(clients)
        {
            clientinfo &ci = *clients[i];
            ci.overflow = 0;
            ci.wsdata = NULL;
            wsmax += ci.position.length();
            if(ci.messages.length()) wsmax += 10 + ci.messages.length();
        }
        if(wsmax <= 0)
        {
            reliablemessages = false;
            return false;
        }
        worldstate &ws = worldstates.add();
        ws.setup(2*wsmax);
        int mtu = getservermtu() - 100;
        if(mtu <= 0) mtu = ws.len;
        ucharbuf wsbuf(ws.data, ws.len);
        loopv(clients)
        {
            clientinfo &ci = *clients[i];
            if(ci.state.aitype != AI_NONE) continue;
            addposition(ws, wsbuf, mtu, ci, ci);
            loopvj(ci.bots) addposition(ws, wsbuf, mtu, *ci.bots[j], ci);
        }
        sendpositions(ws, wsbuf);
        loopv(clients)
        {
            clientinfo &ci = *clients[i];
            if(ci.state.aitype != AI_NONE) continue;
            addmessages(ws, wsbuf, mtu, ci, ci);
            loopvj(ci.bots) addmessages(ws, wsbuf, mtu, *ci.bots[j], ci);
        }
        sendmessages(ws, wsbuf);
        reliablemessages = false;
        if(ws.uses) return true;
        ws.cleanup();
        worldstates.drop();
        return false;
    }

    bool sendpackets(bool force)
    {
        if(clients.empty() || (!hasnonlocalclients() && !demorecord)) return false;
        enet_uint32 curtime = enet_time_get()-lastsend;
        if(curtime<33 && !force) return false;
        bool flush = buildworldstate();
        lastsend += curtime - (curtime%33);
        return flush;
    }

    template<class T>
    void sendstate(gamestate &gs, T &p)
    {
        putint(p, gs.lifesequence);
        putint(p, gs.health);
        putint(p, gs.maxhealth);
        putint(p, gs.armour);
        putint(p, gs.armourtype);
        putint(p, gs.gunselect);
        loopi(GUN_PISTOL-GUN_SG+1) putint(p, gs.ammo[GUN_SG+i]);
    }

    void spawnstate(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        gs.spawnstate(gamemode);
        gs.lifesequence = (gs.lifesequence + 1)&0x7F;
    }

    void sendspawn(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        spawnstate(ci);
        sendf(ci->ownernum, 1, "rii7v", N_SPAWNSTATE, ci->clientnum, gs.lifesequence,
              gs.health, gs.maxhealth,
              gs.armour, gs.armourtype,
              gs.gunselect, GUN_PISTOL-GUN_SG+1, &gs.ammo[GUN_SG]);
        gs.lastspawn = gamemillis;
    }

    bool is_insta(int mode)
    {
        return (gamemodes[mode - STARTGAMEMODE].flags & M_INSTA) != 0;
    }

    void sendwelcome(clientinfo *ci)
    {
        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
        int chan = welcomepacket(p, ci);
        sendpacket(ci->clientnum, chan, p.finalize());
    }

    void putinitclient(clientinfo *ci, packetbuf &p)
    {
        if(ci->state.aitype != AI_NONE)
        {
            putint(p, N_INITAI);
            putint(p, ci->clientnum);
            putint(p, ci->ownernum);
            putint(p, ci->state.aitype);
            putint(p, ci->state.skill);
            putint(p, ci->playermodel);
            sendstring(ci->name, p);
            sendstring(ci->team, p);
        }
        else
        {
            putint(p, N_INITCLIENT);
            putint(p, ci->clientnum);
            sendstring(ci->name, p);
            sendstring(ci->team, p);
            putint(p, ci->playermodel);
        }
    }

    void welcomeinitclient(packetbuf &p, int exclude = -1)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(!ci->connected || ci->clientnum == exclude) continue;
            putinitclient(ci, p);
        }
    }

    bool hasmap(clientinfo *ci)
    {
        return (m_edit && (clients.length() > 0 || ci->local)) ||
        (smapname[0] && (!m_timed || gamemillis < gamelimit || (ci->state.state==CS_SPECTATOR && !ci->privilege && !ci->local) || numclients(ci->clientnum, true, true, true)));
    }

    int welcomepacket(packetbuf &p, clientinfo *ci)
    {
        putint(p, N_WELCOME);
        putint(p, N_MAPCHANGE);
        sendstring(smapname, p);
        putint(p, gamemode);
        putint(p, notgotitems ? 1 : 0);
        if(!ci || (m_timed && smapname[0]))
        {
            putint(p, N_TIMEUP);
            putint(p, gamemillis < gamelimit && !interm ? max((gamelimit - gamemillis)/1000, 1) : 0);
        }
        if(!notgotitems)
        {
            putint(p, N_ITEMLIST);
            loopv(sents) if(sents[i].spawned)
            {
                putint(p, i);
                putint(p, sents[i].type);
            }
            putint(p, -1);
        }
        bool hasmaster = false;
        if(mastermode != MM_OPEN)
        {
            putint(p, N_CURRENTMASTER);
            putint(p, mastermode);
            hasmaster = true;
        }
        loopv(clients) if(clients[i]->privilege >= PRIV_MASTER)
        {
            if(!hasmaster)
            {
                putint(p, N_CURRENTMASTER);
                putint(p, mastermode);
                hasmaster = true;
            }
            putint(p, clients[i]->clientnum);
            putint(p, clients[i]->privilege);
        }
        if(hasmaster) putint(p, -1);
        if(gamepaused)
        {
            putint(p, N_PAUSEGAME);
            putint(p, 1);
            putint(p, -1);
        }
        if(gamespeed != 100)
        {
            putint(p, N_GAMESPEED);
            putint(p, gamespeed);
            putint(p, -1);
        }
        if(m_teammode)
        {
            putint(p, N_TEAMINFO);
            enumerate(teaminfos, teaminfo, t,
                      if(t.frags) { sendstring(t.team, p); putint(p, t.frags); }
                      );
            sendstring("", p);
        }
        if(ci)
        {
            putint(p, N_SETTEAM);
            putint(p, ci->clientnum);
            sendstring(ci->team, p);
            putint(p, -1);
        }
        if(ci && (m_demo || m_mp(gamemode)) && ci->state.state!=CS_SPECTATOR)
        {
            if(smode && !smode->canspawn(ci, true))
            {
                ci->state.state = CS_DEAD;
                putint(p, N_FORCEDEATH);
                putint(p, ci->clientnum);
                sendf(-1, 1, "ri2x", N_FORCEDEATH, ci->clientnum, ci->clientnum);
            }
            else
            {
                gamestate &gs = ci->state;
                spawnstate(ci);
                putint(p, N_SPAWNSTATE);
                putint(p, ci->clientnum);
                sendstate(gs, p);
                gs.lastspawn = gamemillis;
            }
        }
        if(ci && ci->state.state==CS_SPECTATOR)
        {
            putint(p, N_SPECTATOR);
            putint(p, ci->clientnum);
            putint(p, 1);
            sendf(-1, 1, "ri3x", N_SPECTATOR, ci->clientnum, 1, ci->clientnum);
        }
        if(!ci || clients.length()>1)
        {
            putint(p, N_RESUME);
            loopv(clients)
            {
                clientinfo *oi = clients[i];
                if(ci && oi->clientnum==ci->clientnum) continue;
                putint(p, oi->clientnum);
                putint(p, oi->state.state);
                putint(p, oi->state.frags);
                putint(p, oi->state.flags);
                putint(p, oi->state.deaths);
                putint(p, oi->state.quadmillis);
                sendstate(oi->state, p);
            }
            putint(p, -1);
            welcomeinitclient(p, ci ? ci->clientnum : -1);
        }
        if(smode) smode->initclient(ci, p, true);
        return 1;
    }

    bool restorescore(clientinfo *ci)
    {
        if(ci->local) return false;
        savedscore *sc = findscore(ci, false);
        if(sc)
        {
            sc->restore(ci->state);
            return true;
        }
        return false;
    }

    void sendresume(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        sendf(-1, 1, "ri3i4i6vi", N_RESUME, ci->clientnum, gs.state,
              gs.frags, gs.flags, gs.deaths, gs.quadmillis,
              gs.lifesequence,
              gs.health, gs.maxhealth,
              gs.armour, gs.armourtype,
              gs.gunselect, GUN_PISTOL-GUN_SG+1, &gs.ammo[GUN_SG], -1);
    }

    /*extern void checkvotes(bool force=false);
     extern void loaditems();
     extern void changemap(const char *s, int mode);
     extern void rotatemap(bool);
     extern void forcemap(const char *map, int mode);
     extern void vote(const char *map, int reqmode, int sender);
     */

    void sendinitclient(clientinfo *ci)
    {
        packetbuf p(MAXTRANS, ENET_PACKET_FLAG_RELIABLE);
        putinitclient(ci, p);
        sendpacket(-1, 1, p.finalize(), ci->clientnum);
        if(persist && m_teammode) out(ECHO_ALL, "Persistant teams currently enabled");
    }

    void cutogz(char *s)
    {
        char *ogzp = strstr(s, ".ogz");
        if(ogzp) *ogzp = '\0';
    }
    void getmapfilenames(const char *fname, const char *cname, char *pakname, char *mapname, char *cfgname)
    {
        if(!cname) cname = fname;
        string name;
        copystring(name, cname, 100);
        cutogz(name);
        char *slash = strpbrk(name, "/\\");
        if(slash)
        {
            copystring(pakname, name, slash-name+1);
            copystring(cfgname, slash+1);
        }
        else
        {
            copystring(pakname, "base");
            copystring(cfgname, name);
        }
        if(strpbrk(fname, "/\\")) copystring(mapname, fname);
        else formatstring(mapname)("base/%s", fname);
        cutogz(mapname);
    }
    static void fixent(entity &e, int version)
    {
        if(version <= 10 && e.type >= 7) e.type++;
        if(version <= 12 && e.type >= 8) e.type++;
        if(version <= 14 && e.type >= ET_MAPMODEL && e.type <= 16)
        {
            if(e.type == 16) e.type = ET_MAPMODEL;
            else e.type++;
        }
        if(version <= 20 && e.type >= ET_ENVMAP) e.type++;
        if(version <= 21 && e.type >= ET_PARTICLES) e.type++;
        if(version <= 22 && e.type >= ET_SOUND) e.type++;
        if(version <= 23 && e.type >= ET_SPOTLIGHT) e.type++;
        if(version <= 30 && (e.type == ET_MAPMODEL || e.type == ET_PLAYERSTART)) e.attr1 = (int(e.attr1)+180)%360;
        if(version <= 31 && e.type == ET_MAPMODEL) { int yaw = (int(e.attr1)%360 + 360)%360 + 7; e.attr1 = yaw - yaw%15; }
    }

    void readent(entity &e, char *buf, int ver) //read from disk, and init
    {
        if(ver <= 30) switch(e.type)
        {
            case FLAG:
            case MONSTER:
            case TELEDEST:
            case RESPAWNPOINT:
            case BOX:
            case BARREL:
            case PLATFORM:
            case ELEVATOR:
                e.attr1 = (int(e.attr1)+180)%360;
                break;
        }
        if(ver <= 31) switch(e.type)
        {
            case BOX:
            case BARREL:
            case PLATFORM:
            case ELEVATOR:
                int yaw = (int(e.attr1)%360 + 360)%360 + 7;
                e.attr1 = yaw - yaw%15;
                break;
        }
    }

    bool loadentities(const char *fname, vector<entity> &ents, uint *crc)
    {
        string pakname, mapname, mcfgname, ogzname;
        getmapfilenames(fname, NULL, pakname, mapname, mcfgname);
        formatstring(ogzname)("packages/%s.ogz", mapname);
        path(ogzname);
        stream *f = opengzfile(ogzname, "rb");
        if(!f) return false;
        octaheader hdr;
        if(f->read(&hdr, 7*sizeof(int))!=int(7*sizeof(int))) { conoutf(CON_ERROR, "map %s has malformatted header", ogzname); delete f; return false; } //previously commented?
        lilswap(&hdr.version, 6);
        if(memcmp(hdr.magic, "OCTA", 4) || hdr.worldsize <= 0|| hdr.numents < 0) { conoutf(CON_ERROR, "map %s has malformatted header", ogzname); delete f; return false; }
        if(hdr.version>MAPVERSION) { conoutf(CON_ERROR, "map %s requires a newer version of Cube 2: Sauerbraten", ogzname); delete f; return false; }
        compatheader chdr;
        if(hdr.version <= 28)
        {
            if(f->read(&chdr.lightprecision, sizeof(chdr) - 7*sizeof(int)) != int(sizeof(chdr) - 7*sizeof(int))) { conoutf(CON_ERROR, "map %s has malformatted header", ogzname); delete f; return false; }
        }
        else
        {
            int extra = 0;
            if(hdr.version <= 29) extra++;
            if(f->read(&hdr.blendmap, sizeof(hdr) - (7+extra)*sizeof(int)) != int(sizeof(hdr) - (7+extra)*sizeof(int))) { conoutf(CON_ERROR, "map %s has malformatted header", ogzname); delete f; return false; }
        }

        if(hdr.version <= 28)
        {
            lilswap(&chdr.lightprecision, 3);
            hdr.blendmap = chdr.blendmap;
            hdr.numvars = 0;
            hdr.numvslots = 0;
        }
        else
        {
            lilswap(&hdr.blendmap, 2);
            if(hdr.version <= 29) hdr.numvslots = 0;
            else lilswap(&hdr.numvslots, 1);
        }

        loopi(hdr.numvars)
        {
            int type = f->getchar(), ilen = f->getlil<ushort>();
            f->seek(ilen, SEEK_CUR);
            switch(type)
            {
                case ID_VAR: f->getlil<int>(); break;
                case ID_FVAR: f->getlil<float>(); break;
                case ID_SVAR: { int slen = f->getlil<ushort>(); f->seek(slen, SEEK_CUR); break; }
            }
        }

        string gametype;
        copystring(gametype, "fps");
        bool samegame = true;
        int eif = 0;
        if(hdr.version>=16)
        {
            int len = f->getchar();
            f->read(gametype, len+1);
        }
        if(strcmp(gametype, game::gameident()))
        {
            samegame = false;
            conoutf(CON_WARN, "WARNING: loading map from %s game, ignoring entities except for lights & mapmodels", gametype);
        }
        if(hdr.version>=16)
        {
            eif = f->getlil<ushort>();
            int extrasize = f->getlil<ushort>();
            f->seek(extrasize, SEEK_CUR);
        }

        if(hdr.version<14)
        {
            f->seek(256, SEEK_CUR);
        }
        else
        {
            ushort nummru = f->getlil<ushort>();
            f->seek(nummru*sizeof(ushort), SEEK_CUR);
        }

        loopi(min(hdr.numents, MAXENTS))
        {
            entity &e = ents.add();
            f->read(&e, sizeof(entity));
            lilswap(&e.o.x, 3);
            lilswap(&e.attr1, 5);
            fixent(e, hdr.version);
            if(eif > 0) f->seek(eif, SEEK_CUR);
            if(samegame)
            {
                readent(e, NULL, hdr.version);
            }
            else if(e.type>=ET_GAMESPECIFIC || hdr.version<=14)
            {
                ents.pop();
                continue;
            }
        }

        if(crc)
        {
            f->seek(0, SEEK_END);
            *crc = f->getcrc();
        }

        delete f;

        return true;
    }

    bool z_sendmap(clientinfo *ci, clientinfo *sender = NULL, stream *map = NULL, bool force = false, bool verbose = true)
    {
        if(!map) map = mapdata;
        if(!map) { if(verbose && sender) sendf(sender->clientnum, 1, "ris", N_SERVMSG, "no map to send"); }
        else if(ci->getmap && !force)
        {
            if(verbose && sender) sendf(sender->clientnum, 1, "ris", N_SERVMSG,
                                        ci->clientnum == sender->clientnum ? "already sending map" : tempformatstring("already sending map to %s", colorname(ci)));
        }
        else
        {
            if(verbose) sendservmsgf("[%s is getting the map]", colorname(ci));
            ENetPacket *getmap = sendfile(ci->clientnum, 2, map, "ri", N_SENDMAP);
            if(getmap)
            {
                getmap->freeCallback = freegetmap;
                ci->getmap = getmap;
            }
            ci->needclipboard = totalmillis ? totalmillis : 1;
            return true;
        }
        return false;
    }

    SVAR(mappath, "packages/base");
    bool z_savemap(const char *mname, stream *&file = mapdata)
    {
        if(!file) return false;
        int len = (int)min(file->size(), stream::offset(INT_MAX));
        if(len <= 0 && len > 64<<20) return false;
        uchar *data = new uchar[len];
        if(!data) return false;
        file->seek(0, SEEK_SET);
        file->read(data, len);
        delete file;
        string fname;
        if(mappath[0]) sformatstring(fname, "%s/%s.ogz", mappath, mname);
        else sformatstring(fname, "%s.ogz", mname);
        file = openrawfile(path(fname), "w+b");
        if(file)
        {
            file->write(data, len);
            delete[] data;
            return true;
        }
        else
        {
            file = opentempfile("mapdata", "w+b");
            if(file) file->write(data, len);
            delete[] data;
            return false;
        }
    }
    void dosavemap() {
        z_savemap(smapname, mapdata);
    }

    bool z_loadmap(const char *mname, stream *&data = mapdata)
    {
        string fname;
        if(mappath[0]) sformatstring(fname, "%s/%s.ogz", mappath, mname);
        else sformatstring(fname, "%s.ogz", mname);
        stream *map = openrawfile(path(fname), "rb");
        if(!map) return false;
        stream::offset len = map->size();
        if(len <= 0 || len > 16<<20) { delete map; return false; }
        DELETEP(data);
        data = map;
        return true;
    }
    void loadmap(const char *mname) {
        z_loadmap(mname, mapdata);
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            z_sendmap(ci, NULL, mapdata, true, false);
        }
    }

    void listmaps(int sender)
    {
        vector<char *> files;
        vector<char> line;
        listfiles(mappath, "ogz", files);
        files.sort();
        sendf(sender, 1, "ris", N_SERVMSG, files.length() ? "server map files:" : "server has no map files");
        for(int i = 0; i < files.length();)
        {
            line.setsize(0);
            for(int j = 0; i < files.length() && j < 5; i++, j++)
            {
                if(j) line.add(' ');
                line.put(files[i], strlen(files[i]));
            }
            line.add(0);
            sendf(sender, 1, "ris", N_SERVMSG, line.getbuf());
        }
        files.deletearrays();
    }

    void loaditems()
    {
        resetitems();
        notgotitems = true;
        if(m_edit || !loadentities(smapname, ments, &mcrc)) return; //moved from worldio
        loopv(ments) if(canspawnitem(ments[i].type))
        {
            server_entity se = { NOTUSED, 0, false };
            while(sents.length()<=i) sents.add(se);
            sents[i].type = ments[i].type;
            if(m_mp(gamemode) && delayspawn(sents[i].type)) sents[i].spawntime = spawntime(sents[i].type);
            else sents[i].spawned = true;
        }
        notgotitems = false;
    }
    VAR(defaultgamespeed, 10, 100, 1000);
    extern int mapsucksvotes;
    void changemap(const char *s, int mode)
    {
        pausegame(true);
        requestmap(s, mode);
    }

    VAR(matchlength, 0, 600, 1000);

    void _changemap(const char *s, int mode)
    {
        //can cause excess flood on loop i mapchange for IRC
        stopdemo();
        pausegame(false);
        changegamespeed(defaultgamespeed);
        if(smode) smode->cleanup();
        aiman::clearai();
        gamemode = mode;
        gamemillis = 0;
        if (m_overtime) {
            gamelimit = 15 * 60000;
        } else {
            gamelimit = matchlength * 1000;
        }
        interm = 0;
        nextexceeded = 0;
        copystring(smapname, s);
        loaditems();
        scores.shrink(0);
        teamkills.shrink(0);
        int mapsucksvotes = 0;
        firstblood = false;
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            ci->state.timeplayed += lastmillis - ci->state.lasttimeplayed;
            ci->votedmapsucks = false;
        }

        if(!m_mp(gamemode)) kicknonlocalclients(DISC_LOCAL);

        sendf(-1, 1, "risii", N_MAPCHANGE, smapname, gamemode, 1);

        clearteaminfo();
        if(m_teammode && !persist) autoteam();

        if(m_capture) smode = &capturemode;
        else if(m_ctf) smode = &ctfmode;
        else if(m_collect) smode = &collectmode;
        else smode = NULL;

        if(m_timed && smapname[0]) sendf(-1, 1, "ri2", N_TIMEUP, gamemillis < gamelimit && !interm ? max((gamelimit - gamemillis)/1000, 1) : 0);
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            ci->mapchange();
            ci->state.lasttimeplayed = lastmillis;
            if(m_mp(gamemode) && ci->state.state!=CS_SPECTATOR) sendspawn(ci);
        }

        aiman::changemap();

        if(m_demo)
        {
            if(clients.length()) setupdemoplayback();
        }
        else if(demonextmatch)
        {
            demonextmatch = false;
            setupdemorecord();
        }
        if(smode) smode->setup();
        if(autodemo) setupdemorecord();
        loopv(clients)
        {
            if(instacoop) z_loadmap(smapname, mapdata);
            clientinfo *ci = clients[i];
            if(m_edit && autosendmap && enableautosendmap) {
                z_sendmap(ci, NULL, mapdata, true, false);
                if(autosendmap) enableautosendmap = true;
                else enableautosendmap = false;
            }
        }

        healthy();
    }
    ICOMMAND(changemap, "si", (char *target, int *mode), { changemap(target, *mode); });
    ICOMMAND(emptymap, "", (), { changemap("", 1); });
    ICOMMAND(setmode, "i", (int *mode), { changemap(smapname, *mode); });
    ICOMMAND(setmap, "s", (char *target), { changemap(target, gamemode); });

    void rotatemap(bool next)
    {
        if(!maprotations.inrange(curmaprotation))
        {
            changemap("", 1);
            return;
        }
        if(next)
        {
            curmaprotation = findmaprotation(gamemode, smapname);
            if(curmaprotation >= 0) nextmaprotation();
            else curmaprotation = smapname[0] ? max(findmaprotation(gamemode, ""), 0) : 0;
        }
        maprotation &rot = maprotations[curmaprotation];
        changemap(rot.map, rot.findmode(gamemode));
    }

    struct votecount
    {
        char *map;
        int mode, count;
        votecount() {}
        votecount(char *s, int n) : map(s), mode(n), count(0) {}
    };

    void checkvotes(bool force = false)
    {
        vector<votecount> votes;
        int maxvotes = 0;
        loopv(clients)
        {
            clientinfo *oi = clients[i];
            if(oi->state.state==CS_SPECTATOR && !oi->privilege && !oi->local) continue;
            if(oi->state.aitype!=AI_NONE) continue;
            maxvotes++;
            if(!m_valid(oi->modevote)) continue;
            votecount *vc = NULL;
            loopvj(votes) if(!strcmp(oi->mapvote, votes[j].map) && oi->modevote==votes[j].mode)
            {
                vc = &votes[j];
                break;
            }
            if(!vc) vc = &votes.add(votecount(oi->mapvote, oi->modevote));
            vc->count++;
        }
        votecount *best = NULL;
        loopv(votes) if(!best || votes[i].count > best->count || (votes[i].count == best->count && rnd(2))) best = &votes[i];
        if(force || (best && best->count > maxvotes/2))
        {
            if(demorecord) enddemorecord();
            if(best && (best->count > (force ? 1 : maxvotes/2)))
            {
                sendservmsg(force ? "\f7vote passed by default" : "\f7vote passed by majority");
                changemap(best->map, best->mode);
            }
            else rotatemap(true);
        }
    }

    void forcemap(const char *map, int mode)
    {
        stopdemo();
        if(!map[0] && !m_check(mode, M_EDIT))
        {
            int idx = findmaprotation(mode, smapname);
            if(idx < 0 && smapname[0]) idx = findmaprotation(mode, "");
            if(idx < 0) return;
            map = maprotations[idx].map;
        }
        if(hasnonlocalclients()) sendservmsgf("\f7Local player forced \f1%s \f7on map \f6%s", modename(mode), map[0] ? map : "[new map]");
        changemap(map, mode);
    }

    void vote(const char *map, int reqmode, int sender)
    {
        clientinfo *ci = getinfo(sender);
        if(!ci || (ci->state.state==CS_SPECTATOR && !ci->privilege && !ci->local) || (!ci->local && !m_mp(reqmode))) return;
        if(!m_valid(reqmode)) return;
        if(!map[0] && !m_check(reqmode, M_EDIT))
        {
            int idx = findmaprotation(reqmode, smapname);
            if(idx < 0 && smapname[0]) idx = findmaprotation(reqmode, "");
            if(idx < 0) return;
            map = maprotations[idx].map;
        }
        if(lockmaprotation && !ci->local && ci->privilege < (lockmaprotation > 1 ? PRIV_ADMIN : PRIV_MASTER) && findmaprotation(reqmode, map) < 0)
        {
            sendf(sender, 1, "ris", N_SERVMSG, "\f3Error: You may not select a different mode/map");
            return;
        }
        copystring(ci->mapvote, map);
        ci->modevote = reqmode;
        if(ci->local || (ci->privilege && mastermode>=MM_VETO))
        {
            if(demorecord) enddemorecord();
            if(!ci->local || hasnonlocalclients())
                sendservmsgf("\f0%s \f7forced \f1%s \f7on map \f6%s", colorname(ci), modename(ci->modevote), ci->mapvote[0] ? ci->mapvote : "[new map]");
            changemap(ci->mapvote, ci->modevote);
        }
        else
        {
            sendservmsgf("\f0%s \f7votes for \f1%s \f7on map \f3%s\f7. (Use \f2\"/<mode> <map>\" \f7to vote)", colorname(ci), modename(reqmode), map[0] ? map : "[new map]");
            checkvotes();
        }
    }

#define _BESTSTAT(stat) \
{ \
best.setsize(0); \
best.add(clients[0]); \
besti = best[0]->state.stat; \
for(int i = 1; i < clients.length(); i++) \
{ \
if(clients[i]->state.stat > besti) \
{ \
best.setsize(0); \
best.add(clients[i]); \
besti = clients[i]->state.stat; \
} \
else if(clients[i]->state.stat == besti) \
{ \
best.add(clients[i]); \
} \
} \
}

    void _printbest(vector<clientinfo *> &best, int besti, char *msg)
    {
        int l = min(best.length(), 3);
        loopi(l)
        {
            concatstring(msg, colorname(best[i]), MAXTRANS);
            if(i + 1 < l) concatstring(msg, ", ", MAXTRANS);
        }
        defformatstring(buf)(" \f1(\f0%i\f1)", besti);
        concatstring(msg, buf, MAXTRANS);
    }

    void printbeststats()
    {
        vector<clientinfo *> best;
        int besti;
        char msg[MAXTRANS];

        static char const * const bestkills = "\f0The Good:\f7";
        static char const * const worstkills = "\f3The Bad:\f7";

        //Intermission statistics
        msg[0] = '\0';

        // frags
        _BESTSTAT(frags);
        if(besti)
        {
            copystring(msg, bestkills, MAXTRANS);
            concatstring(msg, " \f7frags: \f7", MAXTRANS);
            _printbest(best, besti, msg);
        }

        // damage dealt
        _BESTSTAT(damage);
        if(besti > 0)
        {
            if(!msg[0]) copystring(msg, bestkills, MAXTRANS);
            concatstring(msg, " \f7damage dealt: \f7", MAXTRANS);
            _printbest(best, besti, msg);
        }

        // kpd
        best.setsize(0);
        best.add(clients[0]);
        {
            float bestf = float(best[0]->state.frags) / float(max(best[0]->state.deaths, 1));
            for(int i = 1; i < clients.length(); i++)
            {
                float currf = float(clients[i]->state.frags) / float(max(clients[i]->state.deaths, 1));
                if(currf > bestf)
                {
                    best.setsize(0);
                    best.add(clients[i]);
                    bestf = currf;
                }
                else if(currf == bestf)
                {
                    best.add(clients[i]);
                }
            }

            if(bestf >= 0.01f || bestf <= -0.01f)   // non 0
            {
                if(!msg[0]) copystring(msg, bestkills, MAXTRANS);
                concatstring(msg, " \f7kpd: \f7", MAXTRANS);
                int l = min(best.length(), 3);
                loopi(l)
                {
                    concatstring(msg, colorname(best[i]), MAXTRANS);
                    if(i + 1 < l) concatstring(msg, ", ", MAXTRANS);
                }
                defformatstring(buf)(" \f7(\f0%.2f\f7)", bestf);
                concatstring(msg, buf, MAXTRANS);
            }
        }

        // accuracy
        best.setsize(0);
        best.add(clients[0]);
        besti = best[0]->state.damage * 100 / max(best[0]->state.shotdamage, 1);
        for(int i = 1; i < clients.length(); i++)
        {
            int curri = clients[i]->state.damage * 100 / max(clients[i]->state.shotdamage, 1);
            if(curri > besti)
            {
                best.setsize(0);
                best.add(clients[i]);
                besti = curri;
            }
            else if(curri == besti)
            {
                best.add(clients[i]);
            }
        }
        if(besti)
        {
            if(!msg[0]) copystring(msg, bestkills, MAXTRANS);
            concatstring(msg, " \f7accuracy: \f7", MAXTRANS);
            int l = min(best.length(), 3);
            loopi(l)
            {
                concatstring(msg, colorname(best[i]), MAXTRANS);
                if(i + 1 < l) concatstring(msg, ", ", MAXTRANS);
            }
            defformatstring(buf)(" \f7(\f0%i%%\f7)", besti);
            concatstring(msg, buf, MAXTRANS);
        }

        // print out
        if(msg[0]) sendservmsg(msg);

        // Worst kills
        msg[0] = '\0';

        // deaths
        _BESTSTAT(deaths);
        if(besti)
        {
            copystring(msg, worstkills, MAXTRANS);
            concatstring(msg, " \f7deaths: \f7", MAXTRANS);
            _printbest(best, besti, msg);
        }

        // suicides
        _BESTSTAT(_suicides);
        if(besti)
        {
            if(!msg[0]) copystring(msg, worstkills, MAXTRANS);
            concatstring(msg, " \f7suicides: \f7", MAXTRANS);
            _printbest(best, besti, msg);
        }

        // teamkills
        if(m_teammode)
        {
            _BESTSTAT(teamkills);
            if(besti)
            {
                if(!msg[0]) copystring(msg, worstkills, MAXTRANS);
                concatstring(msg, " \f7teamkills: \f7", MAXTRANS);
                _printbest(best, besti, msg);
            }
        }

        // damage wasted
        best.setsize(0);
        best.add(clients[0]);
        besti = best[0]->state.shotdamage-best[0]->state.damage;
        for(int i = 1; i < clients.length(); i++)
        {
            int curri = clients[i]->state.shotdamage-clients[i]->state.damage;
            if(curri > besti)
            {
                best.setsize(0);
                best.add(clients[i]);
                besti = curri;
            }
            else if(curri == besti)
            {
                best.add(clients[i]);
            }
        }
        if(besti > 0)
        {
            if(!msg[0]) copystring(msg, worstkills, MAXTRANS);
            concatstring(msg, " \f7damage wasted: \f7", MAXTRANS);
            _printbest(best, besti, msg);
        }

        // print out
        if(msg[0]) sendservmsg(msg);

        // Print statuses for ctf modes
        if(m_ctf)
        {
            static char const * const bestflags = "\f7The Flags:\f7";
            _BESTSTAT(flags);
            if(besti)
            {
                copystring(msg, bestflags, MAXTRANS);
                concatstring(msg, " \f7scored: \f7", MAXTRANS);
                _printbest(best, besti, msg);
            }
            else msg[0] = 0;

            if(m_hold)
            {
                _BESTSTAT(_stolen);
                if(besti)
                {
                    if(!msg[0]) copystring(msg, bestflags, MAXTRANS);
                    concatstring(msg, " \f7taken: \f7", MAXTRANS);
                    _printbest(best, besti, msg);
                }
            }
            else if(!m_protect)
            {
                _BESTSTAT(_stolen);
                if(besti)
                {
                    if(!msg[0]) copystring(msg, bestflags, MAXTRANS);
                    concatstring(msg, " \f7stolen: \f7", MAXTRANS);
                    _printbest(best, besti, msg);
                }

                _BESTSTAT(_returned);
                if(besti)
                {
                    if(!msg[0]) copystring(msg, bestflags, MAXTRANS);
                    concatstring(msg, " \f7returned: \f7", MAXTRANS);
                    _printbest(best, besti, msg);
                }
            }
            if(msg[0]) sendservmsg(msg);
        }
        else if(m_collect)
        {
            static char const * const bestskulls = "\f7Best skulls:\f7";
            _BESTSTAT(flags);
            if(besti)
            {
                copystring(msg, bestskulls, MAXTRANS);
                concatstring(msg, " \f7scored: \f7", MAXTRANS);
                _printbest(best, besti, msg);
            }
            else msg[0] = 0;
            _BESTSTAT(_stolen);
            if(besti)
            {
                if(!msg[0]) copystring(msg, bestskulls, MAXTRANS);
                concatstring(msg, " \f7stolen: \f7", MAXTRANS);
                _printbest(best, besti, msg);
            }
            _BESTSTAT(_returned);
            if(besti)
            {
                if(!msg[0]) copystring(msg, bestskulls, MAXTRANS);
                concatstring(msg, " \f7returned: \f7", MAXTRANS);
                _printbest(best, besti, msg);
            }
            if(msg[0]) sendservmsg(msg);
        }
    }
    int number_of(char* team)
    {
        int n = 0;
        loopv(clients)
        {
            if ( isteam(clients[i]->team, team) ) n++;
        }
        return n;
    }

    void checkintermission()
    {
        if(gamemillis >= gamelimit && !interm)
        {
            sendf(-1, 1, "ri2", N_TIMEUP, 0);
            if(smode) smode->intermission();
            changegamespeed(100);
            interm = gamemillis + 10000;
            printbeststats();
        }
    }

    extern void forcespectator(clientinfo *ci)
    {
        if(smode) smode->leavegame(ci);
        ci->state.state = CS_SPECTATOR;
        ci->state.timeplayed += lastmillis - ci->state.lasttimeplayed;
        if(!ci->local && (!ci->privilege || ci->warned)) aiman::removeai(ci);
        sendf(-1, 1, "ri3", N_SPECTATOR, ci->clientnum, 1);
    }

    extern void unspectate(clientinfo *ci)
    {
        ci->state.state = CS_DEAD;
        ci->state.respawn();
        ci->state.lasttimeplayed = lastmillis;
        aiman::addclient(ci);
        sendf(-1, 1, "ri3", N_SPECTATOR, ci->clientnum, 0);
        if(!hasmap(ci)) rotatemap(true);
        //should checkmaps
    }

    void startintermission() {
        gamelimit = min(gamelimit, gamemillis);
        checkintermission();
        out(ECHO_IRC, "Intermission started");
    }

    struct spreemsg {
        int frags;
        string msg1, msg2;
    };

    vector <spreemsg> spreemessages;
    ICOMMAND(addspreemsg, "iss", (int *frags, char *msg1, char *msg2), { spreemsg m; m.frags = *frags; copystring(m.msg1, msg1); copystring(m.msg2, msg2); spreemessages.add(m); });
    struct multikillmsg {
        int frags;
        string msg;
    };
    vector <multikillmsg> multikillmessages;
    ICOMMAND(addmultikillmsg, "is", (int *frags, char *msg), { multikillmsg m; m.frags = *frags; copystring(m.msg, msg); multikillmessages.add(m); });

    void dodamage(clientinfo *target, clientinfo *actor, int damage, int gun, const vec &hitpush = vec(0, 0, 0))
    {
        actor->state.guninfo[gun].damage += damage; //adds damage per guninfo
        gamestate &ts = target->state;
        if(notkdamage) {
            if(!isteam(actor->team, target->team))
                ts.dodamage(damage);
        }
        else if(enable_passflag && actor!=target && isteam(actor->team, target->team)) ctfmode.dopassflagsequence(actor,target);
        else if(nodamage) {}
        else ts.dodamage(damage);
        if(target!=actor && !isteam(target->team, actor->team)) actor->state.damage += damage;
        sendf(-1, 1, "ri6", N_DAMAGE, target->clientnum, actor->clientnum, damage, ts.armour, ts.health);
        if(target==actor) target->setpushed();
        else if(!hitpush.iszero())
        {
            ivec v = vec(hitpush).rescale(DNF);
            sendf(ts.health<=0 ? -1 : target->ownernum, 1, "ri7", N_HITPUSH, target->clientnum, gun, damage, v.x, v.y, v.z);
            target->setpushed();
        }
        if(ts.health<=0)
        {
            //QServ longshot and close up kill (y - depth, z - height, x - left/right)
            float x2 = target->state.o.x;     //target shot x
            float x1 = actor->state.o.x;      //actor shot x
            float y2 = target->state.o.y;     //target shot y
            float y1 = actor->state.o.y;      //actor shot y
            float z2 = target->state.o.z;     //target shot z
            float z1 = actor->state.o.z;      //actor shot z

            float d = sqrt(((x2-x1)*(x2-x1))+((y2-y1)*(y2-y1))+((z2-z1)*(z2-z1)));
            int distanceinteger = int(d + 0.5);

            //no teamkills, or weird negative float
            if(d > 700.0 && distanceinteger > 0 && actor != target && actor->state.aitype == AI_NONE) {
                out(ECHO_SERV,"\f0%s \f7got a longshot kill on \f3%s \f7(Distance: \f7%d\f7 feet) with a \f1%s", colorname(actor), colorname(target), distanceinteger, guns[gun].name);
            }
            if(d <= 20.0 && actor != target && actor->state.aitype == AI_NONE) {
                out(ECHO_SERV,"\f0%s \f7got an up close kill on \f3%s \f7with a \f1%s", colorname(actor), colorname(target), (!strcmp(guns[gun].name, "fist" )) ? "chainsaw" : guns[gun].name);
            }

            target->state.deaths++;
            int fragvalue = smode ? smode->fragvalue(target, actor) : (target==actor || isteam(target->team, actor->team) ? -1 : 1);
            actor->state.frags += fragvalue;

            if(fragvalue>0)
            {
                int friends = 0, enemies = 0; // note: friends also includes the fragger
                if(m_teammode) loopv(clients) if(strcmp(clients[i]->team, actor->team)) enemies++; else friends++;
                else { friends = 1; enemies = clients.length()-1; }
                actor->state.effectiveness += fragvalue*friends/float(max(enemies, 1));
                if(totalmillis - actor->state.lastfragmillis < (int64_t)multifragmillis) {
                    actor->state.multifrags++;
                } else {
                    actor->state.multifrags = 1;
                }
                actor->state.lastfragmillis = totalmillis;
            }
            teaminfo *t = m_teammode ? teaminfos.access(actor->team) : NULL;
            if(t) t->frags += fragvalue;
            sendf(-1, 1, "ri5", N_DIED, target->clientnum, actor->clientnum, actor->state.frags, t ? t->frags : 0);
            if(!firstblood && actor != target) { firstblood = true; out(ECHO_SERV, "\f0%s \f7drew \f6FIRST BLOOD!", colorname(actor)); }
            if(actor != target && actor->state.aitype == AI_NONE) actor->state.spreefrags++;
            if(target->state.spreefrags >= minspreefrags && target->state.aitype == AI_NONE) {
                if(actor == target)
                    out(ECHO_SERV, "\f0%s \f7%s", colorname(target), spreesuicidemsg);
                else
                    out(ECHO_SERV, "\f0%s's \f7%s \f6%s", colorname(target), spreefinmsg, colorname(actor));
            }
            target->state.spreefrags = 0;
            target->state.multifrags = 0;
            target->state.lastfragmillis = 0;
            loopv(spreemessages) {
                if(actor->state.spreefrags == spreemessages[i].frags) out(ECHO_SERV, "\f0%s \f7%s \f6%s", colorname(actor), spreemessages[i].msg1, spreemessages[i].msg2);
            }
            target->position.setsize(0);
            if(smode) smode->died(target, actor);
            ts.state = CS_DEAD;
            ts.lastdeath = gamemillis;
            if(actor!=target && isteam(actor->team, target->team))
            {
                actor->state.teamkills++;
                addteamkill(actor, target, 1);
                defformatstring(msg)("\f7Say sorry to \f1%s\f7. You have teamkilled (%d/%d times). You will be banned if you teamkill %d more times.", colorname(target), actor->state.teamkills, maxteamkills, maxteamkills-actor->state.teamkills);
                if(actor->clientnum < 128) sendf(actor->clientnum, 1, "ris", N_SERVMSG, msg); //don't send msg to bots
                defformatstring(srryfrag)("\f7You were teamkilled by: \f0%s \f7(\f3%d\f7). Use \f2#forgive %d \f7or use \f2#callops \f7to make a report.", colorname(actor), actor->state.teamkills, actor->clientnum);
                if(target->clientnum < 128) sendf(target->clientnum, 1, "ris", N_SERVMSG, srryfrag); //don't send msg to bots
                out(ECHO_NOCOLOR, "Teamkiller: %s (%d)", colorname(actor), actor->state.teamkills);
            }
            ts.deadflush = ts.lastdeath + DEATHMILLIS;
            // ts.respawn(); don't issue respawn yet until DEATHMILLIS has elapsed
        }
    }

    void suicide(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        if(gs.state!=CS_ALIVE) return;
        int fragvalue = smode ? smode->fragvalue(ci, ci) : -1;
        ci->state.frags += fragvalue;
        ci->state.deaths++;
        teaminfo *t = m_teammode ? teaminfos.access(ci->team) : NULL;
        if(t) t->frags += fragvalue;
        sendf(-1, 1, "ri5", N_DIED, ci->clientnum, ci->clientnum, gs.frags, t ? t->frags : 0);
        gs.spreefrags = 0;
        gs.multifrags = 0;
        gs.lastfragmillis = 0;
        ci->position.setsize(0);
        if(smode) smode->died(ci, NULL);
        gs.state = CS_DEAD;
        gs.lastdeath = gamemillis;
        gs.respawn();
        ci->state._suicides++;
    }

    // Kill players and respawn them
    void resetplayers(bool resetfrags)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            gamestate &gs = ci->state;
            teaminfo *t = m_teammode ? teaminfos.access(ci->team) : NULL;
            if (resetfrags) {
                ci->state.frags = 0;
                ci->state.deaths = 0;
                if(t) t->frags = 0;
            }
            sendf(-1, 1, "ri5", N_DIED, ci->clientnum, ci->clientnum, gs.frags, t ? t->frags : 0);
            gs.spreefrags = 0;
            gs.multifrags = 0;
            gs.lastfragmillis = 0;
            ci->position.setsize(0);
            if(smode) smode->died(ci, NULL);
            gs.state = CS_DEAD;
            gs.lastdeath = gamemillis;
            gs.respawn();
            ci->state._suicides = 0;
        }
    }
    ICOMMAND(resetplayers, "i", (int val), { resetplayers(val == 1); });

    void forcerespawn(int n)
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if (n != -1 && ci->clientnum != n) continue;
            if(m_mp(gamemode) && ci->state.state!=CS_SPECTATOR) sendspawn(ci);
        }
    }
    ICOMMAND(forcerespawn, "i", (int *val), forcerespawn(*val));

    void refreshserverinfo()
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            // Clients send N_CONNECT whenever they receive N_SERVINFO, so we ignore it
            // This lets us "hack" the server description so that we can update it after
            // they connect
            ci->ignoreconnect = true;
            sendf(ci->clientnum, 1, "ri5ss", N_SERVINFO, ci->clientnum, PROTOCOL_VERSION, ci->sessionid, serverpass[0] ? 1 : 0, serverdesc, serverauth);
        }
    }
    ICOMMAND(refreshserverinfo, "", (), refreshserverinfo());

    void settime(int secs)
    {
        sendf(-1, 1, "ri2", N_TIMEUP, secs);
    }
    ICOMMAND(settime, "i", (int *val), settime(*val));

    void suicideevent::process(clientinfo *ci) { suicide(ci); }

    void explodeevent::process(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        switch(gun)
        {
            case GUN_RL:
                if(!gs.rockets.remove(id)) return;
                break;

            case GUN_GL:
                if(!gs.grenades.remove(id)) return;
                break;

            default:
                return;
        }
        sendf(-1, 1, "ri4x", N_EXPLODEFX, ci->clientnum, gun, id, ci->ownernum);
        loopv(hits)
        {
            hitinfo &h = hits[i];
            clientinfo *target = getinfo(h.target);
            if(!target || target->state.state!=CS_ALIVE || h.lifesequence!=target->state.lifesequence || h.dist<0 || h.dist>guns[gun].exprad) continue;

            bool dup = false;
            loopj(i) if(hits[j].target==h.target) { dup = true; break; }
            if(dup) continue;

            int damage = guns[gun].damage;
            if(gs.quadmillis) damage *= 4;
            damage = int(damage*(1-h.dist/EXP_DISTSCALE/guns[gun].exprad));
            if(target==ci) damage /= EXP_SELFDAMDIV;
            dodamage(target, ci, damage, gun, h.dir);
        }
    }

    void shotevent::process(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        int wait = millis - gs.lastshot;
        if(!gs.isalive(gamemillis) ||
           wait<gs.gunwait ||
           gun<GUN_FIST || gun>GUN_PISTOL ||
           gs.ammo[gun]<=0 || (guns[gun].range && from.dist(to) > guns[gun].range + 1))
            return;
        if(gun!=GUN_FIST) gs.ammo[gun]--;
        gs.lastshot = millis;
        gs.gunwait = guns[gun].attackdelay;
        sendf(-1, 1, "rii9x", N_SHOTFX, ci->clientnum, gun, id,
              int(from.x*DMF), int(from.y*DMF), int(from.z*DMF),
              int(to.x*DMF), int(to.y*DMF), int(to.z*DMF),
              ci->ownernum);
        //adds shotdamage per guninfo
        gs.shotdamage += guns[gun].damage*(gs.quadmillis ? 4 : 1)*guns[gun].rays;
        gs.guninfo[gun].shotdamage += guns[gun].damage*(gs.quadmillis ? 4 : 1)*guns[gun].rays;

        switch(gun)
        {
            case GUN_RL: gs.rockets.add(id); break;
            case GUN_GL: gs.grenades.add(id); break;
            default:
            {
                int totalrays = 0, maxrays = guns[gun].rays;
                loopv(hits)
                {
                    hitinfo &h = hits[i];
                    clientinfo *target = getinfo(h.target);
                    if(!target || target->state.state!=CS_ALIVE || h.lifesequence!=target->state.lifesequence || h.rays<1 || h.dist > guns[gun].range + 1) continue;

                    totalrays += h.rays;
                    if(totalrays>maxrays) continue;
                    int damage = h.rays*guns[gun].damage;
                    if(gs.quadmillis) damage *= 4;
                    dodamage(target, ci, damage, gun, h.dir);
                }
                break;
            }
        }
    }

    void pickupevent::process(clientinfo *ci)
    {
        gamestate &gs = ci->state;
        if(m_mp(gamemode) && !gs.isalive(gamemillis)) return;
        pickup(ent, ci->clientnum);

    }

    bool gameevent::flush(clientinfo *ci, int fmillis)
    {
        process(ci);
        return true;
    }

    bool timedevent::flush(clientinfo *ci, int fmillis)
    {
        if(millis > fmillis) return false;
        else if(millis >= ci->lastevent)
        {
            ci->lastevent = millis;
            process(ci);
        }
        return true;
    }

    void clearevent(clientinfo *ci)
    {
        delete ci->events.remove(0);
    }

    void flushevents(clientinfo *ci, int millis)
    {
        while(ci->events.length())
        {
            gameevent *ev = ci->events[0];
            if(ev->flush(ci, millis)) clearevent(ci);
            else break;
        }
    }

    void processevents()
    {
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(curtime>0 && ci->state.quadmillis) ci->state.quadmillis = max(ci->state.quadmillis-curtime, 0);
            flushevents(ci, gamemillis);
        }
    }

    //QServ Banner
    vector<char *> banners;
    ICOMMAND(addbanner, "s", (char *text), {
        banners.add(newstring(text));
    });

    VAR(bannerintervolmillis, 1000, 10000, 500000);
    void updateBanner()
    {
        static int lastshow = lastmillis;
        if(banners.length() > 0)
        {
            if((lastmillis-lastshow) >= bannerintervolmillis)
            {
                defformatstring(text)("%s", banners[rnd(banners.length())]);
                sendservmsg(text);
                lastshow = lastmillis;
            } else return;
        }
    }

    void cleartimedevents(clientinfo *ci)
    {
        int keep = 0;
        loopv(ci->events)
        {
            if(ci->events[i]->keepable())
            {
                if(keep < i)
                {
                    for(int j = keep; j < i; j++) delete ci->events[j];
                    ci->events.remove(keep, i - keep);
                    i = keep;
                }
                keep = i+1;
                continue;
            }
        }
        while(ci->events.length() > keep) delete ci->events.pop();
        ci->timesync = false;
    }

    void serverupdate()
    {
        if(shouldstep && !gamepaused)
        {
            gamemillis += curtime;

            if(m_demo) readdemo();
            else if(!m_timed || gamemillis < gamelimit)
            {
                processevents();
                if(curtime)
                {
                    updateBanner();
                    loopv(sents) if(sents[i].spawntime) //spawn entities when timer reached
                    {
                        int oldtime = sents[i].spawntime;
                        sents[i].spawntime -= curtime;
                        if(sents[i].spawntime<=0)
                        {
                            sents[i].spawntime = 0;
                            sents[i].spawned = true;
                            sendf(-1, 1, "ri2", N_ITEMSPAWN, i);
                        }
                        else if(sents[i].spawntime<=10000 && oldtime>10000 && (sents[i].type==I_QUAD || sents[i].type==I_BOOST))
                        {
                            sendf(-1, 1, "ri2", N_ANNOUNCE, sents[i].type);
                        }
                    }
                }
                aiman::checkai();
                if(smode) smode->update();
            }
        }

        while(bannedips.length() && bannedips[0].expire-totalmillis <= 0) bannedips.remove(0);
        loopv(connects) if(totalmillis-connects[i]->connectmillis>15000) disconnect_client(connects[i]->clientnum, DISC_TIMEOUT);
        if(nextexceeded && gamemillis > nextexceeded && (!m_timed || gamemillis < gamelimit))
        {
            nextexceeded = 0;
            loopvrev(clients)
            {
                clientinfo &c = *clients[i];
                if(c.state.aitype != AI_NONE) continue;
                if(c.checkexceeded()) disconnect_client(c.clientnum, DISC_MSGERR);
                else c.scheduleexceeded();
            }
        }
        checkteamkills();

        if(shouldstep && !gamepaused)
        {
            if(m_timed && smapname[0] && gamemillis-curtime>0) checkintermission();
            if(interm > 0 && gamemillis>interm)
            {
                if(demorecord) enddemorecord();
                interm = -1;
                checkvotes(true);
            }
        }
       	//multi kill
        loopv(clients) {
            clientinfo *ci = clients[i];
            if(totalmillis - ci->state.lastfragmillis >= (int64_t)multifragmillis) {
                if(ci->state.multifrags >= minmultikill) {
                    char *msg = NULL;
                    loopv(multikillmessages) {
                        if(multikillmessages[i].frags == ci->state.multifrags) {
                            msg = multikillmessages[i].msg;
                            break;
                        }
                    }
                    if(msg) out(ECHO_SERV,"\f2%s scored a \f6%s", colorname(ci), msg);
                    else out(ECHO_SERV,"\f2%s scored a \f6%s (%d)", colorname(ci), defmultikillmsg, ci->state.multifrags);
                }
                ci->state.multifrags = 0;
            }
        }
        shouldstep = clients.length() > 0;
    }
    struct crcinfo
    {
        int crc, matches;

        crcinfo() {}
        crcinfo(int crc, int matches) : crc(crc), matches(matches) {}

        static bool compare(const crcinfo &x, const crcinfo &y) { return x.matches > y.matches; }
    };

    void privilegemsg(int min_privilege, const char *fmt, ...) {
        va_list ap;
        va_start(ap, fmt);
        loopv(clients) if(clients[i]->privilege >= min_privilege && clients[i]) vmessage(clients[i]->clientnum, fmt, ap);
        va_end(ap);
    }

    void checkmaps(int req = -1)
    {
        if(m_edit || !smapname[0]) return;
        vector<crcinfo> crcs;
        int total = 0, unsent = 0, invalid = 0;
        if(mcrc) crcs.add(crcinfo(mcrc, clients.length() + 1));
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->state.state==CS_SPECTATOR || ci->state.aitype != AI_NONE) continue;
            total++;
            if(!ci->clientmap[0])
            {
                if(ci->mapcrc < 0) invalid++;
                else if(!ci->mapcrc) unsent++;
            }
            else
            {
                crcinfo *match = NULL;
                loopvj(crcs) if(crcs[j].crc == ci->mapcrc) { match = &crcs[j]; break; }
                if(!match) crcs.add(crcinfo(ci->mapcrc, 1));
                else match->matches++;
            }
        }
        if(!mcrc && total - unsent < min(total, 4)) return;
        crcs.sort(crcinfo::compare);
        string msg;
        loopv(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->state.state==CS_SPECTATOR || ci->state.aitype != AI_NONE || ci->clientmap[0] || ci->mapcrc >= 0 || (req < 0 && ci->warned)) continue;
            formatstring(msg)("\f3[Warning]: \f0%s \f7has a modified map file \"%s\")", colorname(ci), smapname);
            sendf(req, 1, "ris", N_SERVMSG, msg);
            out(ECHO_NOCOLOR,"[Warning]: %s has modified map %s", colorname(ci), smapname);
            if(req < 0) ci->warned = true;
        }
        if(crcs.empty() || crcs.length() < 2) return;
        loopv(crcs)
        {
            crcinfo &info = crcs[i];
            if(i || info.matches <= crcs[i+1].matches) loopvj(clients)
            {
                clientinfo *ci = clients[j];
                if(ci->state.state==CS_SPECTATOR || ci->state.aitype != AI_NONE || !ci->clientmap[0] || ci->mapcrc != info.crc || (req < 0 && ci->warned)) continue;
                formatstring(msg)("\f3[Warning]: \f0%s \f7has a modified map file \"%s\")", colorname(ci), smapname);
                sendf(req, 1, "ris", N_SERVMSG, msg);
                out(ECHO_NOCOLOR,"[Warning]: %s has modified map %s", colorname(ci), smapname);
                if(req < 0) ci->warned = true;
            }
        }
    }

    void sendservinfo(clientinfo *ci)
    {
        sendf(ci->clientnum, 1, "ri5ss", N_SERVINFO, ci->clientnum, PROTOCOL_VERSION, ci->sessionid, serverpass[0] ? 1 : 0, serverdesc, serverauth);
    }

    void noclients()
    {
        if(clearbansonempty) bannedips.shrink(0);
        aiman::clearai();
        if(_newflagrun) {_storeflagruns(); _newflagrun = 0;}
        changegamespeed(defaultgamespeed); //return back to normal gamespeed
    }

    void localconnect(int n)
    {
        clientinfo *ci = getinfo(n);
        privilegemsg(PRIV_MASTER, "\f7Connected from host IP");
        ci->clientnum = ci->ownernum = n;
        ci->connectmillis = totalmillis;
        ci->sessionid = (rnd(0x1000000)*((totalmillis%10000)+1))&0xFFFFFF;
        ci->local = true;
        connects.add(ci);
        sendservinfo(ci);
    }

    void localdisconnect(int n)
    {
        clientinfo *ci = getinfo(n);
        if(m_demo) enddemoplayback();
        clientdisconnect(n);
    }

    int clientconnect(int n, uint ip, char *ipstr)
    {
        if(getvar("serverconnectmsg")) {privilegemsg(PRIV_MASTER, "\f7Client detected...");}
        clientinfo *ci = getinfo(n);
        ci->ip=ipstr; //QServ ci->ip
        ci->clientnum = ci->ownernum = n;
        ci->connectmillis = totalmillis;
        ci->sessionid = (rnd(0x1000000)*((totalmillis%10000)+1))&0xFFFFFF;
        connects.add(ci);
        if(ci->isSpecLocked) forcespectator(ci);
        if(ci->votedmapsucks) ci->votedmapsucks = true;
        if(!m_mp(gamemode)) return DISC_LOCAL;
        sendservinfo(ci);
        return DISC_NONE;
    }
    void clientdisconnect(int n)
    {
        clientinfo *ci = getinfo(n);
        if (!ci) return;
        loopv(clients) if(clients[i]->authkickvictim == ci->clientnum) clients[i]->cleanauth();
        if(ci->connected)
        {
            int numofclients = numclients(-1, true, true);
            if(ci->privilege && !ci->isInvAdmin) setmaster(ci, false);
            if(smode) smode->leavegame(ci, true);
            ci->state.timeplayed += lastmillis - ci->state.lasttimeplayed;
            savescore(ci);
            sendf(-1, 1, "ri2", N_CDIS, n);
            clients.removeobj(ci);
            aiman::removeai(ci);
            if(clearbansonempty) {
                if(!numclients(-1, false, true)) noclients(); //define noclients
            }
            if(ci->local) checkpausegame();

            qs.resetoLangWarn(ci->clientnum);
        }
        else connects.removeobj(ci);
    }

    int reserveclients() { return 3; }

    extern void verifybans();

    struct banlist
    {
        vector<ipmask> bans;
        void clear() { bans.shrink(0); }

        bool check(uint ip)
        {
            loopv(bans) if(bans[i].check(ip)) return true;
            return false;
        }

        void add(const char *ipname)
        {
            ipmask ban;
            ban.parse(ipname);
            bans.add(ban);
            verifybans();
        }
    } ipbans, gbans;

    bool checkbans(uint ip)
    {
        loopv(bannedips) if(bannedips[i].ip==ip) return true;
        return ipbans.check(ip) || gbans.check(ip);
    }

    void verifybans()
    {
        loopvrev(clients)
        {
            clientinfo *ci = clients[i];
            if(ci->state.aitype != AI_NONE || ci->local || ci->privilege >= PRIV_ADMIN) continue;
            if(checkbans(getclientip(ci->clientnum))) disconnect_client(ci->clientnum, DISC_IPBAN);
        }
    }

    void ipban(const char *ipname) {
        ipbans.add(ipname);
    }

    void clearpbans() {
        ipbans.clear();
    }

    ICOMMAND(clearpbans, "", (), ipbans.clear());
    ICOMMAND(ipban, "s", (const char *ipname), ipbans.add(ipname));

    void sendkickbanlist(int cn)
    {
        string buf;
        vector<char> msgbuf;
        ipmask im;
        im.mask = ~0;
        int n;
        sendf(cn, 1, "ris", N_SERVMSG, bannedips.empty() ? "kick/ban list is empty" : "kick/ban list:");
        loopv(bannedips)
        {
            msgbuf.setsize(0);
            im.ip = bannedips[i].ip;
            n = sprintf(buf, "\f2id:\f7%2d\f2, ip: \f7", i);
            msgbuf.put(buf, n);
            n = im.print(buf);
            msgbuf.put(buf, n);
            n = sprintf(buf, "\f2, expires in: \f7");
            msgbuf.put(buf, n);
            formatsecs(msgbuf, (uint)((bannedips[i].expire-totalmillis)/1000));
            if(bannedips[i].reason)
            {
                n = snprintf(buf, sizeof(buf), "\f2, reason: \f7%s", bannedips[i].reason);
                msgbuf.put(buf, clamp(n, 0, int(sizeof(buf)-1)));
            }
            msgbuf.add(0);
            sendf(cn, 1, "ris", N_SERVMSG, msgbuf.getbuf());
        }
    }

    void unkickban(int banid, int sender)
    {
        if(bannedips.inrange(banid) && banid >= 0)
        {
            bannedips.remove(banid);
            sendf(sender, 1, "ris", N_SERVMSG, "kick/ban removed");
            banid = banid+banid++; //the kick/ban is now gone so occupy the id by another one
        }
        else if(banid < 0) {
            bannedips.shrink(0);
            sendservmsg("cleared all kicks/bans");
        }
        else {
            sendf(sender, 1, "ris", N_SERVMSG, tempformatstring("Invalid kick/ban id: %d", banid));
            return;
        }
    }

    int allowconnect(clientinfo *ci, const char *pwd = "")
    {
        if(ci->local) return DISC_NONE;
        if(!m_mp(gamemode)) return DISC_LOCAL;
        if(serverpass[0])
        {
            if(!checkpassword(ci, serverpass, pwd)) return DISC_PASSWORD;
            return DISC_NONE;
        }
        if(adminpass[0] && checkpassword(ci, adminpass, pwd)) return DISC_NONE;
        if(numclients(-1, false, true)>=maxclients) return DISC_MAXCLIENTS;
        uint ip = getclientip(ci->clientnum);
        if(checkbans(ip)) return DISC_IPBAN;
        if(mastermode>=MM_PRIVATE && allowedips.find(ip)<0) return DISC_PRIVATE;
        return DISC_NONE;
    }

    bool allowbroadcast(int n)
    {
        clientinfo *ci = getinfo(n);
        return ci && ci->connected;
    }

    clientinfo *findauth(uint id)
    {
        loopv(clients) if(clients[i]->authreq == id) return clients[i];
        return NULL;
    }

    void authfailed(uint id)
    {
        clientinfo *ci = findauth(id);
        if(!ci) return;
        ci->cleanauth();
        if(ci->connectauth) disconnect_client(ci->clientnum, ci->connectauth);
    }

    void authsucceeded(uint id)
    {
        clientinfo *ci = findauth(id);
        if(!ci) return;
        ci->cleanauth(ci->connectauth!=0);
        if(ci->connectauth) connected(ci);
        if(ci->authkickvictim >= 0)
        {
            if(setmaster(ci, true, "", ci->authname, NULL, PRIV_AUTH, false, true))
                trykick(ci, ci->authkickvictim, ci->authkickreason, ci->authname, NULL, PRIV_AUTH);
            ci->cleanauthkick();
        }
        else setmaster(ci, true, "", ci->authname, NULL, PRIV_AUTH);
    }

    void authchallenged(uint id, const char *val, const char *desc = "")
    {
        clientinfo *ci = findauth(id);
        if(!ci) return;
        sendf(ci->clientnum, 1, "risis", N_AUTHCHAL, desc, id, val);
    }

    uint nextauthreq = 0;

    bool tryauth(clientinfo *ci, const char *user, const char *desc)
    {
        ci->cleanauth();
        if(!nextauthreq) nextauthreq = 1;
        ci->authreq = nextauthreq++;
        //filtertext(ci->authname, user, false, false, 100); //variable removed
        filtertext(ci->authname, user, false, 100);
        copystring(ci->authdesc, desc);
        if(ci->authdesc[0])
        {
            userinfo *u = users.access(userkey(ci->authname, ci->authdesc));
            if(u)
            {
                uint seed[3] = { ::hthash(serverauth) + detrnd(size_t(ci) + size_t(user) + size_t(desc), 0x10000), uint(totalmillis), randomMT() };
                vector<char> buf;
                ci->authchallenge = genchallenge(u->pubkey, seed, sizeof(seed), buf);
                sendf(ci->clientnum, 1, "risis", N_AUTHCHAL, desc, ci->authreq, buf.getbuf());
            }
            else ci->cleanauth();
        }
        else if(!requestmasterf("reqauth %u %s\n", ci->authreq, ci->authname))
        {
            ci->cleanauth();
            sendf(ci->clientnum, 1, "ris", N_SERVMSG, "not connected to authentication server");
        }
        if(ci->authreq) return true;
        if(ci->connectauth) disconnect_client(ci->clientnum, ci->connectauth);
        return false;
    }

    void answerchallenge(clientinfo *ci, uint id, char *val, const char *desc)
    {
        if(ci->authreq != id || strcmp(ci->authdesc, desc))
        {
            ci->cleanauth();
            if(ci->connectauth) disconnect_client(ci->clientnum, ci->connectauth);
            return;
        }
        for(char *s = val; *s; s++)
        {
            if(!isxdigit(*s)) { *s = '\0'; break; }
        }
        if(desc[0])
        {
            if(ci->authchallenge && checkchallenge(val, ci->authchallenge))
            {
                userinfo *u = users.access(userkey(ci->authname, ci->authdesc));
                if(u)
                {
                    if(ci->connectauth) connected(ci);
                    if(ci->authkickvictim >= 0)
                    {
                        if(setmaster(ci, true, "", ci->authname, ci->authdesc, u->privilege, false, true))
                            trykick(ci, ci->authkickvictim, ci->authkickreason, ci->authname, ci->authdesc, u->privilege);
                    }
                    else setmaster(ci, true, "", ci->authname, ci->authdesc, u->privilege);
                }
            }
            ci->cleanauth();
        }
        else if(!requestmasterf("confauth %u %s\n", id, val))
        {
            ci->cleanauth();
            sendf(ci->clientnum, 1, "ris", N_SERVMSG, "not connected to authentication server");
        }
        if(!ci->authreq && ci->connectauth) disconnect_client(ci->clientnum, ci->connectauth);
    }

    void processmasterinput(const char *cmd, int cmdlen, const char *args)
    {
        uint id;
        string val;
        if(sscanf(cmd, "failauth %u", &id) == 1)
            authfailed(id);
        else if(sscanf(cmd, "succauth %u", &id) == 1)
            authsucceeded(id);
        else if(sscanf(cmd, "chalauth %u %255s", &id, val) == 2)
            authchallenged(id, val);
        else if(matchstring(cmd, cmdlen, "cleargbans"))
            gbans.clear();
        else if(sscanf(cmd, "addgban %100s", val) == 1)
            gbans.add(val);

    }

    void receivefile(int sender, uchar *data, int len)
    {
        clientinfo *ci = getinfo(sender);
        if(!m_edit || len <= 0 || len > 4*1024*1024 || instacoop && ci->privilege != PRIV_ADMIN) return; //ignore empty sendmaps/instacoop w/o admin
        if(ci->state.state==CS_SPECTATOR && !ci->privilege && !ci->local) return;
        if(mapdata) DELETEP(mapdata);
        //if(!len) return; unneeded empty file check
        mapdata = opentempfile("mapdata", "w+b");
        if(!mapdata) {sendf(sender, 1, "ris", N_SERVMSG, "\f3Error: Failed to open temporary file for map"); return;}
        mapdata->write(data, len);
        sendservmsgf("\f0%s \f7uploaded a map to the server, type \f2\"/getmap\" \f7to receive it", colorname(ci));
    }

    void sendclipboard(clientinfo *ci)
    {
        if(!ci->lastclipboard || !ci->clipboard) return;
        bool flushed = false;
        loopv(clients)
        {
            clientinfo &e = *clients[i];
            if(e.clientnum != ci->clientnum && e.needclipboard - ci->lastclipboard >= 0)
            {
                if(!flushed) { flushserver(true); flushed = true; }
                sendpacket(e.clientnum, 1, ci->clipboard);
            }
        }
    }

    void connected(clientinfo *ci)
    {
        if(m_demo) enddemoplayback();

        //if(!hasmap(ci)) rotatemap(false);

        shouldstep = true;

        connects.removeobj(ci);
        clients.add(ci);

        ci->connectauth = 0;
        ci->connected = true;
        ci->ignoreconnect = false;
        ci->needclipboard = totalmillis ? totalmillis : 1;
        if(mastermode>=MM_LOCKED) ci->state.state = CS_SPECTATOR;
        ci->state.lasttimeplayed = lastmillis;

        const char *worst = m_teammode ? chooseworstteam(NULL, ci) : NULL;
        copystring(ci->team, worst ? worst : "good", MAXTEAMLEN+1);

        sendwelcome(ci);
        if(restorescore(ci)) sendresume(ci);
        sendinitclient(ci);

        aiman::addclient(ci);

        if(m_demo) setupdemoplayback();

        if(instacoop) { z_loadmap(smapname, mapdata); ci->isEditMuted = true; }
        if(m_edit && autosendmap && enableautosendmap) {
            z_sendmap(ci, NULL, mapdata, true, false);
            if(autosendmap) enableautosendmap = true;
            else enableautosendmap = false;
        }

        if(servermotd[0]) {
            if(welcomewithname) {
                defformatstring(welcomemsg)("\f7Welcome to %s\f7, \f0%s\f7. %s",serverdesc,colorname(ci),servermotd);
                sendf(ci->clientnum,1,"ris",N_SERVMSG,welcomemsg);
            }
            else {
                defformatstring(welcomenonamemsg)("%s",servermotd); //simple motd
                sendf(ci->clientnum,1,"ris",N_SERVMSG,welcomenonamemsg);
            }
        }
        //qs.getLocation(ci);
        connect_client(ci->clientnum);
        setclientname(ci->clientnum, ci->name);
    }

    int vmessage(int cn, const char *fmt, va_list ap) {
        if(cn >= 0 && !allowbroadcast(cn)) return 0;
        char buf[1024]; //bigger than 'string'
        int r = vsnprintf(buf, 1024, fmt, ap);
        sendf(cn, 1, "ris", N_SERVMSG, buf);
        return r;
    }

    void parsepacket(int sender, int chan, packetbuf &p) //has to parse exactly each byte of the packet
    {
        if(m_teammode) q_teammode = true;
        else if(!m_teammode) q_teammode = false;
        if(instacoop && gamemillis >= instacoop_gamelimit && !interm) startintermission(); //instacoop intermission initializer
        if(sender<0 || p.packet->flags&ENET_PACKET_FLAG_UNSEQUENCED || chan > 2) return;
        char text[MAXTRANS];
        int type;
        clientinfo *ci = sender>=0 ? getinfo(sender) : NULL, *cq = ci, *cm = ci;
        if(ci && !ci->connected)
        {
            if(chan==0) return;
            else if(chan!=1) { disconnect_client(sender, DISC_MSGERR); return; }
            else while(p.length() < p.maxlen) switch(checktype(getint(p), ci))
            {
                case N_CONNECT:
                {
                    getstring(text, p);
                    //filtertext given another variable
                    filtertext(text, text, false, MAXNAMELEN);
                    if(!text[0]) copystring(text, "unnamed");
                    copystring(ci->name, text, MAXNAMELEN+1);
                    ci->playermodel = getint(p);

                    string password, authdesc, authname;
                    getstring(password, p, sizeof(password));
                    getstring(authdesc, p, sizeof(authdesc));
                    getstring(authname, p, sizeof(authname));
                    int disc = allowconnect(ci, password);
                    if(disc)
                    {
                        if(disc == DISC_LOCAL || !serverauth[0] || strcmp(serverauth, authdesc) || !tryauth(ci, authname, authdesc))
                        {
                            disconnect_client(sender, disc);
                            return;
                        }
                        ci->connectauth = disc;
                    }
                    else
                        connected(ci);
                    break;
                }
                case N_AUTHANS:
                {
                    string desc, ans;
                    getstring(desc, p, sizeof(desc));
                    uint id = (uint)getint(p);
                    getstring(ans, p, sizeof(ans));
                    answerchallenge(ci, id, ans, desc);
                    break;
                }
                case N_PING:
                    getint(p);
                    break;

                default:
                    disconnect_client(sender, DISC_MSGERR);
                    break;
            }
            return;
        }
        else if(chan==2)
        {
            receivefile(sender, p.buf, p.maxlen);
            return;
        }

        if(p.packet->flags&ENET_PACKET_FLAG_RELIABLE) reliablemessages = true;
#define QUEUE_AI clientinfo *cm = cq;
#define QUEUE_MSG { if(cm && (!cm->local || demorecord || hasnonlocalclients())) while(curmsg<p.length()) cm->messages.add(p.buf[curmsg++]); }
#define QUEUE_BUF(body) { \
if(cm && (!cm->local || demorecord || hasnonlocalclients())) \
{ \
curmsg = p.length(); \
{ body; } \
} \
}
#define QUEUE_INT(n) QUEUE_BUF(putint(cm->messages, n))
#define QUEUE_UINT(n) QUEUE_BUF(putuint(cm->messages, n))
#define QUEUE_STR(text) QUEUE_BUF(sendstring(text, cm->messages))

        int curmsg;
        int ct = 0;

        if (!ci) return;

        while((curmsg = p.length()) < p.maxlen) {
            int type = type = checktype(getint(p), ci);
            if (false) {
                conoutf("client -> server %s", gettype(type));
            }
            switch(type) {
            case N_POS:
            {
                int pcn = getuint(p);
                p.get();
                uint flags = getuint(p);
                clientinfo *cp = getinfo(pcn);
                if(cp && pcn != sender && cp->ownernum != sender) cp = NULL;
                vec pos;
                loopk(3)
                {
                    int n = p.get(); n |= p.get()<<8; if(flags&(1<<k)) { n |= p.get()<<16; if(n&0x800000) n |= -1<<24; }
                    pos[k] = n/DMF;
                }
                loopk(3) p.get();
                int mag = p.get(); if(flags&(1<<3)) mag |= p.get()<<8;
                int dir = p.get(); dir |= p.get()<<8;
                vec vel = vec((dir%360)*RAD, (clamp(dir/360, 0, 180)-90)*RAD).mul(mag/DVELF);
                if(flags&(1<<4))
                {
                    p.get(); if(flags&(1<<5)) p.get();
                    if(flags&(1<<6)) loopk(2) p.get();
                }
                if(cp)
                {
                    if((!ci->local || demorecord || hasnonlocalclients()) && (cp->state.state==CS_ALIVE || cp->state.state==CS_EDITING))
                    {
                        if(!ci->local && !m_edit && max(vel.magnitude2(), (float)fabs(vel.z)) >= 180)
                            cp->setexceeded();
                        cp->position.setsize(0);
                        while(curmsg<p.length()) cp->position.add(p.buf[curmsg++]);
                    }
                    if(smode && cp->state.state==CS_ALIVE) smode->moved(cp, cp->state.o, cp->gameclip, pos, (flags&0x80)!=0);
                    cp->state.o = pos;
                    cp->gameclip = (flags&0x80)!=0;
                }
                break;
            }

            case N_TELEPORT:
            {
                int pcn = getint(p), teleport = getint(p), teledest = getint(p);
                clientinfo *cp = getinfo(pcn);
                if(cp && pcn != sender && cp->ownernum != sender) cp = NULL;
                if(cp && (!ci->local || demorecord || hasnonlocalclients()) && (cp->state.state==CS_ALIVE || cp->state.state==CS_EDITING))
                {
                    flushclientposition(*cp);
                    sendf(-1, 0, "ri4x", N_TELEPORT, pcn, teleport, teledest, cp->ownernum);
                }
                break;
            }

            case N_CONNECT:
            {
                if (!ci->ignoreconnect) {
                    disconnect_client(sender, DISC_MSGERR);
                    return;
                }
                getstring(text, p);
                getint(p);
                string password, authdesc, authname;
                getstring(password, p, sizeof(password));
                getstring(authdesc, p, sizeof(authdesc));
                getstring(authname, p, sizeof(authname));
                break;
            }

            case N_JUMPPAD:
            {
                int pcn = getint(p), jumppad = getint(p);
                clientinfo *cp = getinfo(pcn);
                if(cp && pcn != sender && cp->ownernum != sender) cp = NULL;
                if(cp && (!ci->local || demorecord || hasnonlocalclients()) && (cp->state.state==CS_ALIVE || cp->state.state==CS_EDITING))
                {
                    cp->setpushed();
                    flushclientposition(*cp);
                    sendf(-1, 0, "ri3x", N_JUMPPAD, pcn, jumppad, cp->ownernum);
                }
                break;
            }

            case N_FROMAI:
            {
                int qcn = getint(p);
                if(qcn < 0) cq = ci;
                else
                {
                    cq = getinfo(qcn);
                    if(cq && qcn != sender && cq->ownernum != sender) cq = NULL;
                }
                break;
            }

            case N_EDITMODE:
            {
                int val = getint(p);
                if(!ci->local && !m_edit) break;
                if(val ? ci->state.state!=CS_ALIVE && ci->state.state!=CS_DEAD : ci->state.state!=CS_EDITING) break;
                if(smode)
                {
                    if(val) smode->leavegame(ci);
                    else smode->entergame(ci);
                }
                if(val)
                {
                    if(instacoop) {
                        forcespectator(ci);
                        ci->isSpecLocked = true;
                        sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Editing is not allowed, please reconnect to continue playing.");
                    }
                    else {
                        ci->state.editstate = ci->state.state;
                        ci->state.state = CS_EDITING;
                    }
                    ci->events.setsize(0);
                    ci->state.rockets.reset();
                    ci->state.grenades.reset();

                }
                else ci->state.state = ci->state.editstate;
                QUEUE_MSG;
                break;
            }

            case N_MAPCRC:
            {
                getstring(text, p);
                int crc = getint(p);
                if(!ci) break;
                if(strcmp(text, smapname))
                {
                    if(ci->clientmap[0])
                    {
                        ci->clientmap[0] = '\0';
                        ci->mapcrc = 0;
                    }
                    else if(ci->mapcrc > 0) ci->mapcrc = 0;
                    break;
                }
                copystring(ci->clientmap, text);
                ci->mapcrc = text[0] ? crc : 1;
                checkmaps();
                break;
            }

            case N_CHECKMAPS:
                checkmaps(sender);
                break;

            case N_TRYSPAWN:
                if(!ci || !cq || cq->state.state!=CS_DEAD || cq->state.lastspawn>=0 || (smode && !smode->canspawn(cq))) break;
                if(!ci->clientmap[0] && !ci->mapcrc)
                {
                    ci->mapcrc = -1;
                    checkmaps();
                }
                if(cq->state.deadflush)
                {
                    flushevents(cq, cq->state.deadflush);
                    cq->state.respawn();
                }
                cleartimedevents(cq);
                sendspawn(cq);
                break;

            case N_GUNSELECT:
            {
                int gunselect = getint(p);
                if(!cq || cq->state.state!=CS_ALIVE) break;
                cq->state.gunselect = gunselect >= GUN_FIST && gunselect <= GUN_PISTOL ? gunselect : GUN_FIST;
                QUEUE_AI;
                QUEUE_MSG;
                break;
            }

            case N_SPAWN:
            {
                int ls = getint(p), gunselect = getint(p);
                if(!cq || (cq->state.state!=CS_ALIVE && cq->state.state!=CS_DEAD) || ls!=cq->state.lifesequence || cq->state.lastspawn<0) break;
                cq->state.lastspawn = -1;
                cq->state.state = CS_ALIVE;
                cq->state.gunselect = gunselect >= GUN_FIST && gunselect <= GUN_PISTOL ? gunselect : GUN_FIST;
                cq->exceeded = 0;
                if(smode) smode->spawned(cq);
                QUEUE_AI;
                QUEUE_BUF({
                    putint(cm->messages, N_SPAWN);
                    sendstate(cq->state, cm->messages);
                });
                break;
            }

            case N_SUICIDE:
            {
                if(cq) cq->addevent(new suicideevent);
                break;
            }

            case N_SHOOT:
            {
                shotevent *shot = new shotevent;
                shot->id = getint(p);
                shot->millis = cq ? cq->geteventmillis(gamemillis, shot->id) : 0;
                shot->gun = getint(p);
                loopk(3) shot->from[k] = getint(p)/DMF;
                loopk(3) shot->to[k] = getint(p)/DMF;
                int hits = getint(p);
                loopk(hits)
                {
                    if(p.overread()) break;
                    hitinfo &hit = shot->hits.add();
                    hit.target = getint(p);
                    hit.lifesequence = getint(p);
                    hit.dist = getint(p)/DMF;
                    hit.rays = getint(p);
                    loopk(3) hit.dir[k] = getint(p)/DNF;
                }
                if(cq)
                {
                    cq->addevent(shot);
                    cq->setpushed();
                }
                else delete shot;
                break;
            }

            case N_EXPLODE:
            {
                explodeevent *exp = new explodeevent;
                int cmillis = getint(p);
                exp->millis = cq ? cq->geteventmillis(gamemillis, cmillis) : 0;
                exp->gun = getint(p);
                exp->id = getint(p);
                int hits = getint(p);
                loopk(hits)
                {
                    if(p.overread()) break;
                    hitinfo &hit = exp->hits.add();
                    hit.target = getint(p);
                    hit.lifesequence = getint(p);
                    hit.dist = getint(p)/DMF;
                    hit.rays = getint(p);
                    loopk(3) hit.dir[k] = getint(p)/DNF;
                }
                if(cq) cq->addevent(exp);
                else delete exp;
                break;
            }

            case N_ITEMPICKUP:
            {
                int n = getint(p);
                if(!cq) break;
                pickupevent *pickup = new pickupevent;
                pickup->ent = n;
                cq->addevent(pickup);
                break;
            }

            case N_TEXT:
            {
                getstring(text, p);
                filtertext(text, text, true);
                if(totalmillis - ci->lasttext < (int64_t)spammillis) {
                    ci->spamlines++;
                    if(ci->spamlines >= maxspam) {
                        defformatstring(blockedmsginfo)("\f3[Overflow Protection]: \"%s\" was blocked",text);
                        if(!ci->spamwarned) sendf(sender, 1, "ris", N_SERVMSG, blockedmsginfo);
                        ci->spamwarned = true;
                        break;
                    }
                } else {
                    ci->spamwarned = false;
                    ci->spamlines = 0;
                }
                ci->lasttext = totalmillis;
                if(text[0] == '#') {
                    char *c = text;
                    while(*c && isspace(*c)) c++;
                    if(!qs.handleTextCommands(ci, text)) break;
                }
                else {
                    for (int a=0; a<(strlen(*blkmsg)-1); a++) {
                        textblk(blkmsg[a], text, ci);
                    }
                    if(!ci->isMuted) {
                        QUEUE_AI;
                        QUEUE_INT(N_TEXT);
                        QUEUE_STR(text);
                    }
                    else sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Failed to send message: you are muted.");
                }
#include <stdio.h>
#include <time.h>
                struct tm newtime;
                time_t ltime;
                char buf[50];
                ltime=time(&ltime);
                localtime_r(&ltime, &newtime); //localtime__r and asctime_r: threadsafe
                logoutf("%s %s: %s", asctime_r(&newtime, buf), colorname(cq),text);
                out(ECHO_IRC, "%s: %s", colorname(cq),text);
                break;
            }

            case N_SAYTEAM:
            {
                getstring(text, p);
                if(!ci || !cq || (ci->state.state==CS_SPECTATOR && !ci->local && !ci->privilege) || !m_teammode || !cq->team[0]) break;
                filtertext(text, text, true, true);
                loopv(clients)
                {
                    clientinfo *t = clients[i];
                    if(t==cq || t->state.state==CS_SPECTATOR || t->state.aitype != AI_NONE || strcmp(cq->team, t->team)) continue;
                    sendf(t->clientnum, 1, "riis", N_SAYTEAM, cq->clientnum, text);
                }
                logoutf("%s <%s>: %s", colorname(cq), cq->team, text);
                break;
            }

            case N_SWITCHNAME:
            {
                QUEUE_MSG;
                getstring(text, p);
                filtertext(ci->name, text, false, MAXNAMELEN);
                if(!ci->name[0]) copystring(ci->name, "unnamed");
                QUEUE_STR(ci->name);
                out(ECHO_NOCOLOR, "%s changed their name", text);
                setclientname(ci->clientnum, ci->name);
                break;
            }

            case N_SWITCHMODEL:
            {
                ci->playermodel = getint(p);
                QUEUE_MSG;
                break;
            }

            case N_SWITCHTEAM:
            {
                getstring(text, p);
                filtertext(text, text, false, MAXTEAMLEN);
                if(m_teammode && text[0] && strcmp(ci->team, text) && (!smode || smode->canchangeteam(ci, ci->team, text)) && addteaminfo(text))
                {
                    if(ci->state.state==CS_ALIVE) suicide(ci);
                    copystring(ci->team, text);
                    aiman::changeteam(ci);
                    sendf(-1, 1, "riisi", N_SETTEAM, sender, ci->team, ci->state.state==CS_SPECTATOR ? -1 : 0);
                }
                break;
            }

            case N_MAPVOTE:
            {
                getstring(text, p);
                filtertext(text, text, false);
                int reqmode = getint(p);
                vote(text, reqmode, sender);
                break;
            }

            case N_ITEMLIST:
            {
                if((ci->state.state==CS_SPECTATOR && !ci->privilege && !ci->local) || !notgotitems || strcmp(ci->clientmap, smapname)) { while(getint(p)>=0 && !p.overread()) getint(p); break; }
                int n;
                while((n = getint(p))>=0 && n<MAXENTS && !p.overread())
                {
                    server_entity se = { NOTUSED, 0, false };
                    while(sents.length()<=n) sents.add(se);
                    sents[n].type = getint(p);
                    if(canspawnitem(sents[n].type))
                    {
                        if(m_mp(gamemode) && delayspawn(sents[n].type)) sents[n].spawntime = spawntime(sents[n].type);
                        else sents[n].spawned = true;
                    }
                }
                notgotitems = false;
                break;
            }

                //Editmute
            case N_EDITF:   //maptitle, fpush
            case N_EDITM:   //model
            case N_FLIP:    //flipcube
            case N_ROTATE:  //rotate
            case N_DELCUBE: //editdelcube
            {
                int size = server::msgsizelookup(type);
                if(size<=0) { disconnect_client(sender, DISC_MSGERR); return; }
                loopi(size-1) getint(p);
                if(ci->isEditMuted)
                {
                    sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3[Warning] Your editing is muted");
                    break;
                }
                else {
                    sendeditmessage(sender, &p, curmsg);
                    QUEUE_AI;
                    QUEUE_MSG;
                    break;
                }
            }

            case N_EDITENT:
            {
                int i = getint(p);
                loopk(3) getint(p);
                int type = getint(p);
                loopk(5) getint(p);
                if(!ci || ci->state.state==CS_SPECTATOR || ci->isEditMuted) break;
                sendeditmessage(sender, &p, curmsg);
                QUEUE_MSG;
                bool canspawn = canspawnitem(type);
                if(i<MAXENTS && (sents.inrange(i) || canspawnitem(type)))
                {
                    server_entity se = { NOTUSED, 0, false };
                    while(sents.length()<=i) sents.add(se);
                    sents[i].type = type;
                    if(canspawn ? !sents[i].spawned : (sents[i].spawned || sents[i].spawntime))
                    {
                        sents[i].spawntime = canspawn ? 1 : 0;
                        sents[i].spawned = false;
                    }
                }
                break;
            }

            case N_EDITVAR:
            {
                int type = getint(p);
                getstring(text, p);
                switch(type)
                {
                    case ID_VAR: getint(p); break;
                    case ID_FVAR: getfloat(p); break;
                    case ID_SVAR: getstring(text, p);
                }
                if(ci && ci->state.state!=CS_SPECTATOR && !ci->isEditMuted) {
                    sendeditmessage(sender, &p, curmsg);
                    QUEUE_MSG;
                }
                else {
                    sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3[Warning] Your variable editing is muted");
                    break;
                }
                break;
            }

            case N_PING:
                sendf(sender, 1, "i2", N_PONG, getint(p));
                break;

            case N_CLIENTPING:
            {
                int ping = getint(p);
                if(ci)
                {
                    ci->ping = ping;
                    loopv(ci->bots) {ci->bots[i]->ping = ping;}
                    if((ci->ping)>getvar("maxpingwarn")) {
                        if(!ci->pingwarned) {
                            if(ci->clientnum < 128) {
                                defformatstring(s)("\f2[Notice]: \f7%s", pingwarncustommsg);
                                sendf(ci->clientnum, 1, "ris", N_SERVMSG, s);
                            }
                            ci->pingwarned = true;
                        }
                    }
                }
                QUEUE_MSG;
                break;
            }

            case N_MASTERMODE:
            {
                int mm = getint(p);
                if((ci->privilege || ci->local) && mm>=MM_OPEN && mm<=MM_PRIVATE)
                {
                    if((ci->privilege>=PRIV_ADMIN || ci->local) || (mastermask&(1<<mm)))
                    {
                        if(mm==MM_PRIVATE && numclients(-1, false, false) == 1 && ci->privilege != PRIV_ADMIN && no_single_private) {
                            defformatstring(s)("\f3Error: Server can't be set to private when only one client is connected.");
                            sendf(sender, 1, "ris", N_SERVMSG, s);
                        }
                        else {
                            mastermode = mm;
                            allowedips.shrink(0);
                            if(mm>=MM_PRIVATE)
                            {
                                loopv(clients) allowedips.add(getclientip(clients[i]->clientnum));
                            }
                            sendf(-1, 1, "rii", N_MASTERMODE, mastermode);
                            out(ECHO_NOCOLOR, "Mastermode: %s (%d)", mastermodename(mastermode), mastermode);
                        }
                    }
                    else
                    {
                        defformatstring(s)("\f3Error: Failed to change mastermode (%d is disabled)", mm);
                        sendf(sender, 1, "ris", N_SERVMSG, s);
                    }
                }
                break;
            }

            case N_CLEARBANS:
            {
                if(ci->privilege || ci->local)
                {
                    bannedips.shrink(0);
                    out(ECHO_SERV, "\f0Server bans cleared!");
                    out(ECHO_CONSOLE, "Server bans cleared!");
                    out(ECHO_IRC, "Server bans cleared!");
                }
                break;
            }

            case N_KICK:
            {
                int victim = getint(p);
                getstring(text, p);
                filtertext(text, text);
                trykick(ci, victim, text);
                teamkillkickreset();
                break;
            }

            case N_SPECTATOR:
            {
                int spectator = getint(p), val = getint(p);
                if(!ci->privilege && !ci->local && (spectator!=sender || (ci->state.state==CS_SPECTATOR && mastermode>=MM_LOCKED))) break;
                clientinfo *spinfo = (clientinfo *)getclientinfo(spectator); //no bots
                if(!spinfo || (spinfo->state.state==CS_SPECTATOR ? val : !val)) break;

                if(spinfo->state.state!=CS_SPECTATOR && val)
                {
                    if(spinfo->state.state==CS_ALIVE) suicide(spinfo);
                    if(smode) smode->leavegame(spinfo);
                    spinfo->state.state = CS_SPECTATOR;
                    spinfo->state.timeplayed += lastmillis - spinfo->state.lasttimeplayed;
                    if(!spinfo->local && !spinfo->privilege) aiman::removeai(spinfo);
                    if(!spinfo->isSpecLocked) out(ECHO_SERV,"\f0%s \f7is now a spectator", colorname(spinfo));
                }
                else if(spinfo->state.state==CS_SPECTATOR && !val)
                {
                    spinfo->state.state = CS_DEAD;
                    spinfo->state.respawn();
                    spinfo->state.lasttimeplayed = lastmillis;
                    aiman::addclient(spinfo);
                    if(spinfo->clientmap[0] || spinfo->mapcrc) checkmaps();
                    if(!spinfo->isSpecLocked) out(ECHO_SERV,"\f0%s \f7is no longer a spectator", colorname(spinfo));

                }
                if(!spinfo->isSpecLocked) {
                    sendf(-1, 1, "ri3", N_SPECTATOR, spectator, val);
                } else {
                    sendf(spinfo->clientnum, 1, "ris", N_SERVMSG, "\f3You are locked in spectator mode.");
                }
                if(!val && !hasmap(spinfo)) rotatemap(true);
                break;
            }

            case N_SETTEAM:
            {
                int who = getint(p);
                getstring(text, p);
                filtertext(text, text, false, MAXTEAMLEN);
                if(!ci->privilege && !ci->local) break;
                clientinfo *wi = getinfo(who);
                if(!m_teammode || !text[0] || !wi || !strcmp(wi->team, text)) break;
                if((!smode || smode->canchangeteam(wi, wi->team, text)) && addteaminfo(text))
                {
                    if(wi->state.state==CS_ALIVE) suicide(wi);
                    copystring(wi->team, text, MAXTEAMLEN+1);
                }
                aiman::changeteam(wi);
                sendf(-1, 1, "riisi", N_SETTEAM, who, wi->team, 1);
                break;
            }

            case N_FORCEINTERMISSION:
                if(ci->local && !hasnonlocalclients()) startintermission();
                break;

            case N_RECORDDEMO:
            {
                int val = getint(p);
                if(ci->privilege < (restrictdemos ? PRIV_ADMIN : PRIV_MASTER) && !ci->local) break;
                if(!maxdemos || !maxdemosize)
                {
                    sendf(ci->clientnum, 1, "ris", N_SERVMSG, "\f3Error: Demo recording disabled");
                    break;
                }
                demonextmatch = val!=0;
                sendservmsgf("\f7demo recording is %s \f7for next match", demonextmatch ? "\f0enabled" : "\f3disabled");
                break;
            }

            case N_STOPDEMO:
            {
                if(ci->privilege < (restrictdemos ? PRIV_ADMIN : PRIV_MASTER) && !ci->local) break;
                stopdemo();
                break;
            }

            case N_CLEARDEMOS:
            {
                int demo = getint(p);
                if(ci->privilege < (restrictdemos ? PRIV_ADMIN : PRIV_MASTER) && !ci->local) break;
                cleardemos(demo);
                break;
            }

            case N_LISTDEMOS:
                if(!ci->privilege && !ci->local && ci->state.state==CS_SPECTATOR) break;
                listdemos(sender);
                break;

            case N_GETDEMO:
            {
                int n = getint(p), tag = getint(p);
                if(!ci->privilege && !ci->local && ci->state.state==CS_SPECTATOR) break;
                senddemo(ci, n, tag);
                break;
            }

            case N_GETMAP:
                if(!mapdata) sendf(sender, 1, "ris", N_SERVMSG, "\f3Error: no map to send");
                else if(ci->getmap) sendf(sender, 1, "ris", N_SERVMSG, "\f7Map is already downloading, please wait.");
                else
                {
                    sendservmsgf("\f0%s \f7is downloading map \"%s\"...", colorname(ci), smapname[0] == '\0' ? "[untitled]" : smapname);
                    if((ci->getmap = sendfile(sender, 2, mapdata, "ri", N_SENDMAP)))
                        ci->getmap->freeCallback = freegetmap;
                    ci->needclipboard = totalmillis ? totalmillis : 1;
                }
                break;


            case N_NEWMAP:
            {
                int size = getint(p);
                if(!ci->privilege && !ci->local && ci->state.state==CS_SPECTATOR) break;
                if(size>=0)
                {
                    smapname[0] = '\0';
                    resetitems();
                    notgotitems = false;
                    if(smode) smode->newmap();
                }
                if(!ci->isEditMuted) {
                    sendeditmessage(sender, &p, curmsg);
                    QUEUE_MSG;
                }
                else {
                    sendf(sender, 1, "ris", N_SERVMSG, "\f3Error: You may not create a new map while you are edit muted.");
                    break;
                }
                break;
            }

            case N_SETMASTER:
            {
                int mn = getint(p), val = getint(p);
                getstring(text, p);
                if(mn != ci->clientnum)
                {
                    if(!ci->privilege && !ci->local) break;
                    clientinfo *minfo = (clientinfo *)getclientinfo(mn);
                    if(!minfo || (!ci->local && minfo->privilege >= ci->privilege) || (val && minfo->privilege)) break;
                    setmaster(minfo, val!=0, "", NULL, NULL, PRIV_MASTER, true);
                }
                else setmaster(ci, val!=0, text);
                // don't broadcast the master password
                break;
            }

            case N_ADDBOT:
            {
                aiman::reqadd(ci, getint(p));
                break;
            }

            case N_DELBOT:
            {
                aiman::reqdel(ci);
                break;
            }

            case N_BOTLIMIT:
            {
                int limit = getint(p);
                if(ci) aiman::setbotlimit(ci, limit);
                break;
            }

            case N_BOTBALANCE:
            {
                int balance = getint(p);
                if(ci) aiman::setbotbalance(ci, balance!=0);
                break;
            }

            case N_AUTHTRY:
            {
                string desc, name;
                getstring(desc, p, sizeof(desc));
                getstring(name, p, sizeof(name));
                tryauth(ci, name, desc);
                break;
            }

            case N_AUTHKICK:
            {
                string desc, name;
                getstring(desc, p, sizeof(desc));
                getstring(name, p, sizeof(name));
                int victim = getint(p);
                getstring(text, p);
                filtertext(text, text);
                int authpriv = PRIV_AUTH;
                if(desc[0])
                {
                    userinfo *u = users.access(userkey(name, desc));
                    if(u) authpriv = u->privilege; else break;
                }
                if(trykick(ci, victim, text, name, desc, authpriv, true) && tryauth(ci, name, desc))
                {
                    ci->authkickvictim = victim;
                    ci->authkickreason = newstring(text);
                }
                break;
            }

            case N_AUTHANS:
            {
                string desc, ans;
                getstring(desc, p, sizeof(desc));
                uint id = (uint)getint(p);
                getstring(ans, p, sizeof(ans));
                answerchallenge(ci, id, ans, desc);
                break;
            }

            case N_PAUSEGAME:
            {
                int val = getint(p);
                if(ci->privilege < (restrictpausegame ? PRIV_ADMIN : PRIV_MASTER) && !ci->local) break;
                pausegame(val > 0, ci);
                break;
            }

            case N_GAMESPEED:
            {
                int val = getint(p);
                if(ci->privilege < (restrictgamespeed ? PRIV_ADMIN : PRIV_MASTER) && !ci->local) break;
                changegamespeed(val, ci);
                break;
            }

            case N_COPY:
                ci->cleanclipboard();
                ci->lastclipboard = totalmillis ? totalmillis : 1;
                sendeditmessage(sender, &p, curmsg);
                goto genericmsg;

            case N_PASTE:
                if(ci->state.state!=CS_SPECTATOR) sendclipboard(ci);
                sendeditmessage(sender, &p, curmsg);
                goto genericmsg;

            case N_CLIPBOARD:
            {
                int unpacklen = getint(p), packlen = getint(p);
                ci->cleanclipboard(false);
                if(ci->state.state==CS_SPECTATOR)
                {
                    if(packlen > 0) p.subbuf(packlen);
                    break;
                }
                if(packlen <= 0 || packlen > (1<<16) || unpacklen <= 0)
                {
                    if(packlen > 0) p.subbuf(packlen);
                    packlen = unpacklen = 0;
                }
                packetbuf q(32 + packlen, ENET_PACKET_FLAG_RELIABLE);
                putint(q, N_CLIPBOARD);
                putint(q, ci->clientnum);
                putint(q, unpacklen);
                putint(q, packlen);
                if(packlen > 0) p.get(q.subbuf(packlen).buf, packlen);
                ci->clipboard = q.finalize();
                ci->clipboard->referenceCount++;
                break;
            }

            case N_EDITT:
            case N_REPLACE:
            case N_EDITVSLOT:
            {
                int size = server::msgsizelookup(type);
                if(size<=0) { disconnect_client(sender, DISC_MSGERR); return; }
                loopi(size-1) getint(p);
                if(p.remaining() < 2) { disconnect_client(sender, DISC_MSGERR); return; }
                int extra = lilswap(*(const ushort *)p.pad(2));
                if(p.remaining() < extra) { disconnect_client(sender, DISC_MSGERR); return; }
                p.pad(extra);
                if(ci && ci->state.state!=CS_SPECTATOR) {
                    sendeditmessage(sender, &p, curmsg);
                    QUEUE_MSG;
                }
                break;
            }

            case N_UNDO:
            case N_REDO:
            {
                int unpacklen = getint(p), packlen = getint(p);
                if(!ci || ci->state.state==CS_SPECTATOR || packlen <= 0 || packlen > (1<<16) || unpacklen <= 0)
                {
                    if(packlen > 0) p.subbuf(packlen);
                    break;
                }
                if(p.remaining() < packlen) { disconnect_client(sender, DISC_MSGERR); return; }
                packetbuf q(32 + packlen, ENET_PACKET_FLAG_RELIABLE);
                putint(q, type);
                putint(q, ci->clientnum);
                putint(q, unpacklen);
                putint(q, packlen);
                if(packlen > 0) p.get(q.subbuf(packlen).buf, packlen);
                sendpacket(-1, 1, q.finalize(), ci->clientnum);
                break;
            }

            case N_SERVCMD:
                getstring(text, p);
                break;

#define PARSEMESSAGES 1
#include "capture.h"
#include "ctf.h"
#include "collect.h"
#undef PARSEMESSAGES

            case -1:
                disconnect_client(sender, DISC_MSGERR);
                return;

            case -2:
                disconnect_client(sender, DISC_OVERFLOW);
                return;

            default: genericmsg:
            {
                int size = server::msgsizelookup(type);
                if(size<=0) { disconnect_client(sender, DISC_MSGERR); return; }
                loopi(size-1) getint(p);
                if(ci && cq && (ci != cq || ci->state.state!=CS_SPECTATOR)) { QUEUE_AI; QUEUE_MSG; }
                break;
            }
        }
        }
    }

    int laninfoport() { return SAUERBRATEN_LANINFO_PORT; }
    int serverinfoport(int servport) { return servport < 0 ? SAUERBRATEN_SERVINFO_PORT : servport+1; }
    int serverport(int infoport) { return infoport < 0 ? SAUERBRATEN_SERVER_PORT : infoport-1; }
    const char *defaultmaster() { return "master.sauerbraten.org"; }
    int masterport() { return SAUERBRATEN_MASTER_PORT; }
    int numchannels() { return 3; }

#include "extinfo.h"
    void serverinforeply(ucharbuf &req, ucharbuf &p)
    {
        if(!getint(req))
        {
            extserverinforeply(req, p);
            return;
        }
        putint(p, numclients(-1, false, true));
        putint(p, gamepaused || gamespeed != 100 ? 7 : 5); //number of attrs following
        putint(p, PROTOCOL_VERSION);

        //generic attributes passed back
        putint(p, gamemode);
        putint(p, m_timed ? max((gamelimit - gamemillis)/1000, 0) : 0);

        putint(p, maxclients);
        putint(p, serverpass[0] ? MM_PASSWORD : (!m_mp(gamemode) ? MM_PRIVATE : (mastermode || mastermask&MM_AUTOAPPROVE ? mastermode : MM_AUTH)));
        if(gamepaused || gamespeed != 100)
        {
            putint(p, gamepaused ? 1 : 0);
            putint(p, gamespeed);
        }
        sendstring(smapname, p);
        sendstring(serverdesc, p);
        sendserverinforeply(p);
    }

    bool servercompatible(char *name, char *sdec, char *map, int ping, const vector<int> &attr, int np)
    {
        return attr.length() && attr[0]==PROTOCOL_VERSION;
    }

#include "aiman.h"
}

