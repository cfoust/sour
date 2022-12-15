#ifndef FPSGAME_H_INCLUDED
#define FPSGAME_H_INCLUDED

#include "game.h"

namespace server {
    struct server_entity            // server side version of "entity" type
    {
        int type;
        int spawntime;
        char spawned;
    };
    
    static const int DEATHMILLIS = 300;
    
    struct gameevent
    {
        virtual ~gameevent() {}
        
        virtual bool flush(clientinfo *ci, int fmillis);
        virtual void process(clientinfo *ci) {}
        
        virtual bool keepable() const { return false; }
    };
    
    struct timedevent : gameevent
    {
        int millis;
        
        bool flush(clientinfo *ci, int fmillis);
    };
    
    struct hitinfo
    {
        int target;
        int lifesequence;
        int rays;
        float dist;
        vec dir;
    };
    
    struct shotevent : timedevent
    {
        int id, gun;
        vec from, to;
        vector<hitinfo> hits;
        
        void process(clientinfo *ci);
    };
    
    struct explodeevent : timedevent
    {
        int id, gun;
        vector<hitinfo> hits;
        
        bool keepable() const { return true; }
        
        void process(clientinfo *ci);
    };
    
    struct suicideevent : gameevent
    {
        void process(clientinfo *ci);
    };
    
    struct pickupevent : gameevent
    {
        int ent;
        void process(clientinfo *ci);
    };
    
    template <int N>
    struct projectilestate
    {
        int projs[N];
        int numprojs;
        
        projectilestate() : numprojs(0) {}
        
        void reset() { numprojs = 0; }
        
        void add(int val)
        {
            if(numprojs>=N) numprojs = 0;
            projs[numprojs++] = val;
        }
        
        bool remove(int val)
        {
            loopi(numprojs) if(projs[i]==val)
            {
                projs[i] = projs[--numprojs];
                return true;
            }
            return false;
        }
    };
    
    struct gamestate : fpsstate
    {
        int64_t lastfragmillis;
        int multifrags;
        int spreefrags;
        vec o;
        int state, editstate;
        int lastdeath, deadflush, lastspawn, lifesequence;
        int lastshot;
        projectilestate<8> rockets, grenades;
        int frags, flags, deaths, teamkills, shotdamage, damage, tokens;
        int _suicides, _stolen, _returned; //stats for QServ
        int lasttimeplayed, timeplayed;
        float effectiveness;
        
        struct
        {
            int shotdamage;
            int damage;
        } guninfo[NUMGUNS];
        
        gamestate() : state(CS_DEAD), editstate(CS_DEAD) {}
        
        bool isalive(int gamemillis)
        {
            return state==CS_ALIVE || (state==CS_DEAD && gamemillis - lastdeath <= DEATHMILLIS);
        }
        
        bool waitexpired(int gamemillis)
        {
            return gamemillis - lastshot >= gunwait;
        }
        
        void reset()
        {
            if(state!=CS_SPECTATOR) state = editstate = CS_DEAD;
            maxhealth = 100;
            rockets.reset();
            grenades.reset();
            
            timeplayed = 0;
            effectiveness = 0;
            frags = flags = deaths = teamkills = shotdamage = damage = tokens = 0;
            lastdeath = 0;
            respawn();
            
            _suicides = _stolen = _returned = 0; //Stats for QServ
            lastfragmillis = 0;
            multifrags = spreefrags = 0;
            
            loopi(NUMGUNS)
            {
                guninfo[i].damage = 0;
                guninfo[i].shotdamage = 0;
            }
            
        }
        
        void respawn()
        {
            fpsstate::respawn();
            o = vec(-1e10f, -1e10f, -1e10f);
            deadflush = 0;
            lastspawn = -1;
            lastshot = 0;
            tokens = 0;
        }
        
        void reassign()
        {
            respawn();
            rockets.reset();
            grenades.reset();
        }
    };
    
    struct savedscore
    {
        uint ip;
        string name;
        int maxhealth, frags, flags, deaths, teamkills, shotdamage, damage;
        int timeplayed;
        float effectiveness;
        
        void save(gamestate &gs)
        {
            maxhealth = gs.maxhealth;
            frags = gs.frags;
            flags = gs.flags;
            deaths = gs.deaths;
            teamkills = gs.teamkills;
            shotdamage = gs.shotdamage;
            damage = gs.damage;
            timeplayed = gs.timeplayed;
            effectiveness = gs.effectiveness;
        }
        
        void restore(gamestate &gs)
        {
            if(gs.health==gs.maxhealth) gs.health = maxhealth;
            gs.maxhealth = maxhealth;
            gs.frags = frags;
            gs.flags = flags;
            gs.deaths = deaths;
            gs.teamkills = teamkills;
            gs.shotdamage = shotdamage;
            gs.damage = damage;
            gs.timeplayed = timeplayed;
            gs.effectiveness = effectiveness;
        }
    };
    
    
    
    extern int gamemillis, nextexceeded;
    
    struct extrainfo
    {
        int lasttakeflag;
    };
    
    struct clientinfo
    {
        char *ip; //ipstring for QServ
        int clientnum, ownernum, connectmillis, sessionid, overflow, connectedmillis; //qserv
        uint sourtype;
        string name, team, mapvote;
        int playermodel;
        int modevote;
        int privilege;
        bool connected, local, timesync;
        int gameoffset, lastevent, pushed, exceeded;
        gamestate state;
        vector<gameevent *> events;
        vector<uchar> position, messages;
        uchar *wsdata;
        int wslen;
        vector<clientinfo *> bots;
        int ping, aireinit;
        string clientmap;
        int mapcrc;
        bool warned, gameclip;
        ENetPacket *getdemo, *getmap, *clipboard;
        int lastclipboard, needclipboard;
        int connectauth;
        uint authreq;
        string authname, authdesc;
        void *authchallenge;
        int authkickvictim;
        char *authkickreason;
        extrainfo _xi; //xi for QServ flagrun stats
        
        /*QServ*/
        bool isMuted = false;
        bool isSpecLocked = false;
        bool isEditMuted = false;
        bool pingwarned = false;
        bool votedmapsucks = false;
        bool isInvAdmin = false;
        
        int64_t lasttext;
        int spamlines;
        bool spamwarned;
        
        clientinfo() : getdemo(NULL), getmap(NULL), clipboard(NULL), authchallenge(NULL), authkickreason(NULL) { reset(); }
        ~clientinfo() { events.deletecontents(); cleanclipboard(); cleanauth(); }
        
        void addevent(gameevent *e)
        {
            if(state.state==CS_SPECTATOR || events.length()>100) delete e;
            else events.add(e);
        }
        
        enum
        {
            PUSHMILLIS = 2500
        };
        
        int calcpushrange()
        {
            ENetPeer *peer = getclientpeer(ownernum);
            return PUSHMILLIS + (peer ? peer->roundTripTime + peer->roundTripTimeVariance : ENET_PEER_DEFAULT_ROUND_TRIP_TIME);
        }
        
        bool checkpushed(int millis, int range)
        {
            return millis >= pushed - range && millis <= pushed + range;
        }
        
        void scheduleexceeded()
        {
            if(state.state!=CS_ALIVE || !exceeded) return;
            int range = calcpushrange();
            if(!nextexceeded || exceeded + range < nextexceeded) nextexceeded = exceeded + range;
        }
        
        void setexceeded()
        {
            if(state.state==CS_ALIVE && !exceeded && !checkpushed(gamemillis, calcpushrange())) exceeded = gamemillis;
            scheduleexceeded();
        }
        
        void setpushed()
        {
            pushed = max(pushed, gamemillis);
            if(exceeded && checkpushed(exceeded, calcpushrange())) exceeded = 0;
        }
        
        bool checkexceeded()
        {
            return state.state==CS_ALIVE && exceeded && gamemillis > exceeded + calcpushrange();
        }
        
        void mapchange()
        {
            mapvote[0] = 0;
            modevote = INT_MAX;
            state.reset();
            events.deletecontents();
            overflow = 0;
            timesync = false;
            lastevent = 0;
            exceeded = 0;
            pushed = 0;
            clientmap[0] = '\0';
            mapcrc = 0;
            warned = false;
            gameclip = false;
            _xi.lasttakeflag = 0;
        }
        
        void reassign()
        {
            state.reassign();
            events.deletecontents();
            timesync = false;
            lastevent = 0;
        }
        
        void cleanclipboard(bool fullclean = true)
        {
            if(clipboard) { if(--clipboard->referenceCount <= 0) enet_packet_destroy(clipboard); clipboard = NULL; }
            if(fullclean) lastclipboard = 0;
        }
        
        void cleanauthkick()
        {
            authkickvictim = -1;
            DELETEA(authkickreason);
        }
        
        void cleanauth(bool full = true)
        {
            authreq = 0;
            if(authchallenge) { freechallenge(authchallenge); authchallenge = NULL; }
            if(full) cleanauthkick();
        }
        
        void reset()
        {
            name[0] = team[0] = 0;
            playermodel = -1;
            privilege = PRIV_NONE;
            connected = local = false;
            connectauth = 0;
            position.setsize(0);
            messages.setsize(0);
            ping = 0;
            aireinit = 0;
            needclipboard = 0;
            cleanclipboard();
            cleanauth();
            mapchange();
            lasttext = spamlines = 0; //QServ Anti message flood
        }
        
        int geteventmillis(int servmillis, int clientmillis)
        {
            if(!timesync || (events.empty() && state.waitexpired(servmillis)))
            {
                timesync = true;
                gameoffset = servmillis - clientmillis;
                return servmillis;
            }
            else return gameoffset + clientmillis;
        }
    };
    
    struct ban
    {
        int time, expire;
        uint ip;
        int type;
        char *reason;   // unique pointer
        ban(): reason(NULL) {}
        ban(const ban &b): reason(NULL) { *this = b; }
        ~ban() { delete[] reason; }
        ban &operator =(const ban &b)
        {
            if(&b != this) {
                time = b.time;
                expire = b.expire;
                ip = b.ip;
                type = b.type;
                delete[] reason;
                reason = b.reason;
                ((ban *)&b)->reason = NULL;    // ugly hack
            }
            return *this;
        }
    };
    
    namespace aiman
    {
        extern void removeai(clientinfo *ci);
        extern void clearai();
        extern void checkai();
        extern void reqadd(clientinfo *ci, int skill);
        extern void reqdel(clientinfo *ci);
        extern void setbotlimit(clientinfo *ci, int limit);
        extern void setbotbalance(clientinfo *ci, bool balance);
        extern void changemap();
        extern void addclient(clientinfo *ci);
        extern void changeteam(clientinfo *ci);
    }
    
#define MM_MODE 0xF
#define MM_AUTOAPPROVE 0x1000
#define MM_PRIVSERV (MM_MODE | MM_AUTOAPPROVE)
#define MM_PUBSERV ((1<<MM_OPEN) | (1<<MM_VETO))
#define MM_COOPSERV (MM_AUTOAPPROVE | MM_PUBSERV | (1<<MM_LOCKED))
    
    extern vector<clientinfo *> connects, clients, bots;
    extern int mastermode;
    
    //QServ
    extern void send_connected_time(clientinfo *ci, int sender);
    extern int vmessage(int cn, const char *fmt, va_list ap);
    extern bool duplicatename(clientinfo *ci, char *name);
    extern const char *colorname(clientinfo *ci);
    extern void revokemaster(clientinfo *ci);
    extern void checkpausegame();
    extern bool setmaster(clientinfo *ci, bool val, const char *pass, const char *authname, const char *authdesc, int authpriv, bool force, bool trial, bool revoke);
    
}

#endif
