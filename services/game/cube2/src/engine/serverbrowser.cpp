// serverbrowser.cpp: eihrul's concurrent resolver, and server browser window management

#include "engine.h"
#if __EMSCRIPTEN__
#include <emscripten.h>
#endif

struct resolverthread
{
    SDL_Thread *thread;
    const char *query;
    int starttime;
};

struct resolverresult
{
    const char *query;
    ENetAddress address;
};

vector<resolverthread> resolverthreads;
vector<const char *> resolverqueries;
vector<resolverresult> resolverresults;
SDL_mutex *resolvermutex;
SDL_cond *querycond, *resultcond;

#define RESOLVERTHREADS 2
#define RESOLVERLIMIT 3000

int resolverloop(void * data)
{
    resolverthread *rt = (resolverthread *)data;
    SDL_LockMutex(resolvermutex);
    SDL_Thread *thread = rt->thread;
    SDL_UnlockMutex(resolvermutex);
    if(!thread || SDL_GetThreadID(thread) != SDL_ThreadID())
        return 0;
    while(thread == rt->thread)
    {
        SDL_LockMutex(resolvermutex);
        while(resolverqueries.empty()) SDL_CondWait(querycond, resolvermutex);
        rt->query = resolverqueries.pop();
        rt->starttime = totalmillis;
        SDL_UnlockMutex(resolvermutex);

        ENetAddress address = { ENET_HOST_ANY, ENET_PORT_ANY };
        enet_address_set_host(&address, rt->query);

        SDL_LockMutex(resolvermutex);
        if(rt->query && thread == rt->thread)
        {
            resolverresult &rr = resolverresults.add();
            rr.query = rt->query;
            rr.address = address;
            rt->query = NULL;
            rt->starttime = 0;
            SDL_CondSignal(resultcond);
        }
        SDL_UnlockMutex(resolvermutex);
    }
    return 0;
}

void resolverinit()
{
    resolvermutex = SDL_CreateMutex();
    querycond = SDL_CreateCond();
    resultcond = SDL_CreateCond();

    SDL_LockMutex(resolvermutex);
    loopi(RESOLVERTHREADS)
    {
        resolverthread &rt = resolverthreads.add();
        rt.query = NULL;
        rt.starttime = 0;
        rt.thread = SDL_CreateThread(resolverloop, "resolver", &rt);
    }
    SDL_UnlockMutex(resolvermutex);
}

void resolverstop(resolverthread &rt)
{
    SDL_LockMutex(resolvermutex);
    if(rt.query)
    {
#if SDL_VERSION_ATLEAST(2, 0, 2)
        SDL_DetachThread(rt.thread);
#endif
        rt.thread = SDL_CreateThread(resolverloop, "resolver", &rt);
    }
    rt.query = NULL;
    rt.starttime = 0;
    SDL_UnlockMutex(resolvermutex);
} 

void resolverclear()
{
    if(resolverthreads.empty()) return;

    SDL_LockMutex(resolvermutex);
    resolverqueries.shrink(0);
    resolverresults.shrink(0);
    loopv(resolverthreads)
    {
        resolverthread &rt = resolverthreads[i];
        resolverstop(rt);
    }
    SDL_UnlockMutex(resolvermutex);
}

void resolverquery(const char *name)
{
    if(resolverthreads.empty()) resolverinit();

    SDL_LockMutex(resolvermutex);
    resolverqueries.add(name);
    SDL_CondSignal(querycond);
    SDL_UnlockMutex(resolvermutex);
}

bool resolvercheck(const char **name, ENetAddress *address)
{
    bool resolved = false;
    SDL_LockMutex(resolvermutex);
    if(!resolverresults.empty())
    {
        resolverresult &rr = resolverresults.pop();
        *name = rr.query;
        address->host = rr.address.host;
        resolved = true;
    }
    else loopv(resolverthreads)
    {
        resolverthread &rt = resolverthreads[i];
        if(rt.query && totalmillis - rt.starttime > RESOLVERLIMIT)        
        {
            resolverstop(rt);
            *name = rt.query;
            resolved = true;
        }    
    }
    SDL_UnlockMutex(resolvermutex);
    return resolved;
}

bool resolverwait(const char *name, ENetAddress *address)
{
    if(resolverthreads.empty()) resolverinit();

    defformatstring(text, "resolving %s... (esc to abort)", name);
    renderprogress(0, text);

    SDL_LockMutex(resolvermutex);
    resolverqueries.add(name);
    SDL_CondSignal(querycond);
    int starttime = SDL_GetTicks(), timeout = 0;
    bool resolved = false;
    for(;;) 
    {
        SDL_CondWaitTimeout(resultcond, resolvermutex, 250);
        loopv(resolverresults) if(resolverresults[i].query == name) 
        {
            address->host = resolverresults[i].address.host;
            resolverresults.remove(i);
            resolved = true;
            break;
        }
        if(resolved) break;
    
        timeout = SDL_GetTicks() - starttime;
        renderprogress(min(float(timeout)/RESOLVERLIMIT, 1.0f), text);
        if(interceptkey(SDLK_ESCAPE)) timeout = RESOLVERLIMIT + 1;
        if(timeout > RESOLVERLIMIT) break;    
    }
    if(!resolved && timeout > RESOLVERLIMIT)
    {
        loopv(resolverthreads)
        {
            resolverthread &rt = resolverthreads[i];
            if(rt.query == name) { resolverstop(rt); break; }
        }
    }
    SDL_UnlockMutex(resolvermutex);
    return resolved && address->host != ENET_HOST_ANY;
}

#define CONNLIMIT 20000

int connectwithtimeout(ENetSocket sock, const char *hostname, const ENetAddress &address)
{
    defformatstring(text, "connecting to %s... (esc to abort)", hostname);
    renderprogress(0, text);

    ENetSocketSet readset, writeset;
    if(!enet_socket_connect(sock, &address)) for(int starttime = SDL_GetTicks(), timeout = 0; timeout <= CONNLIMIT;)
    {
        ENET_SOCKETSET_EMPTY(readset);
        ENET_SOCKETSET_EMPTY(writeset);
        ENET_SOCKETSET_ADD(readset, sock);
        ENET_SOCKETSET_ADD(writeset, sock);
        int result = enet_socketset_select(sock, &readset, &writeset, 250);
        if(result < 0) break;
        else if(result > 0)
        {
            if(ENET_SOCKETSET_CHECK(readset, sock) || ENET_SOCKETSET_CHECK(writeset, sock))
            {
                int error = 0;
                if(enet_socket_get_option(sock, ENET_SOCKOPT_ERROR, &error) < 0 || error) break;
                return 0;
            }
        }
        timeout = SDL_GetTicks() - starttime;
        renderprogress(min(float(timeout)/CONNLIMIT, 1.0f), text);
        if(interceptkey(SDLK_ESCAPE)) break;
    }

    return -1;
}
 
struct pingattempts
{
    enum { MAXATTEMPTS = 2 };

    int offset, attempts[MAXATTEMPTS];

    pingattempts() : offset(0) { clearattempts(); }

    void clearattempts() { memset(attempts, 0, sizeof(attempts)); }

    void setoffset() { offset = 1 + rnd(0xFFFFFF); } 

    int encodeping(int millis)
    {
        millis += offset;
        return millis ? millis : 1;
    }

    int decodeping(int val)
    {
        return val - offset;
    }

    int addattempt(int millis)
    {
        int val = encodeping(millis);
        loopk(MAXATTEMPTS-1) attempts[k+1] = attempts[k];
        attempts[0] = val;
        return val;
    }

    bool checkattempt(int val, bool del = true)
    {
        if(val) loopk(MAXATTEMPTS) if(attempts[k] == val)
        {
            if(del) attempts[k] = 0;
            return true;
        }
        return false;
    }

};

enum { UNRESOLVED = 0, RESOLVING, RESOLVED };

struct serverinfo : pingattempts
{
    enum 
    { 
        WAITING = INT_MAX,

        MAXPINGS = 3
    };

    string name, map, sdesc;
    int port, numplayers, resolved, ping, lastping, nextping;
    int pings[MAXPINGS];
    vector<int> attr;
    ENetAddress address;
    bool keep;
    const char *password;

    serverinfo()
        : port(-1), numplayers(0), resolved(UNRESOLVED), keep(false), password(NULL)
    {
        name[0] = map[0] = sdesc[0] = '\0';
        clearpings();
        setoffset();
    }

    ~serverinfo()
    {
        DELETEA(password);
    }

    void clearpings()
    {
        ping = WAITING;
        loopk(MAXPINGS) pings[k] = WAITING;
        nextping = 0;
        lastping = -1;
        clearattempts();
    }

    void cleanup()
    {
        clearpings();
        attr.setsize(0);
        numplayers = 0;
    }

    void reset()
    {
        lastping = -1;
    }

    void checkdecay(int decay)
    {
        if(lastping >= 0 && totalmillis - lastping >= decay)
            cleanup();
        if(lastping < 0) lastping = totalmillis;
    }

    void calcping()
    {
        int numpings = 0, totalpings = 0;
        loopk(MAXPINGS) if(pings[k] != WAITING) { totalpings += pings[k]; numpings++; }
        ping = numpings ? totalpings/numpings : WAITING;
    }

    void addping(int rtt, int millis)
    {
        if(millis >= lastping) lastping = -1;
        pings[nextping] = rtt;
        nextping = (nextping+1)%MAXPINGS;
        calcping();
    }

    static bool compare(serverinfo *a, serverinfo *b)
    {
        bool ac = server::servercompatible(a->name, a->sdesc, a->map, a->ping, a->attr, a->numplayers),
             bc = server::servercompatible(b->name, b->sdesc, b->map, b->ping, b->attr, b->numplayers);
        if(ac > bc) return true;
        if(bc > ac) return false;
        if(a->keep > b->keep) return true;
        if(a->keep < b->keep) return false;
        if(a->numplayers < b->numplayers) return false;
        if(a->numplayers > b->numplayers) return true;
        if(a->ping > b->ping) return false;
        if(a->ping < b->ping) return true;
        int cmp = strcmp(a->name, b->name);
        if(cmp != 0) return cmp < 0;
        if(a->port < b->port) return true;
        if(a->port > b->port) return false;
        return false;
    }
};

vector<serverinfo *> servers;
ENetSocket pingsock = ENET_SOCKET_NULL;
int lastinfo = 0;

static serverinfo *newserver(const char *name, int port, uint ip = ENET_HOST_ANY)
{
    serverinfo *si = new serverinfo;
    si->address.host = ip;
    si->address.port = server::serverinfoport(port);
    if(ip!=ENET_HOST_ANY) si->resolved = RESOLVED;

    si->port = port;
    if(name) copystring(si->name, name);
    else if(ip==ENET_HOST_ANY || enet_address_get_host_ip(&si->address, si->name, sizeof(si->name)) < 0)
    {
        delete si;
        return NULL;

    }

    servers.add(si);

    return si;
}

void addserver(const char *name, int port, const char *password, bool keep)
{
    if(port <= 0) port = server::serverport();
    loopv(servers)
    {
        serverinfo *s = servers[i];
        if(strcmp(s->name, name) || s->port != port) continue;
        if(password && (!s->password || strcmp(s->password, password)))
        {
            DELETEA(s->password);
            s->password = newstring(password);
        }
        if(keep && !s->keep) s->keep = true;
        return;
    }
    serverinfo *s = newserver(name, port);
    if(!s) return;
    if(password) s->password = newstring(password);
    s->keep = keep;
}

VARP(searchlan, 0, 0, 1);
VARP(servpingrate, 1000, 5000, 60000);
VARP(servpingdecay, 1000, 15000, 60000);
VARP(maxservpings, 0, 10, 1000);

pingattempts lanpings;

template<size_t N> static inline void buildping(ENetBuffer &buf, uchar (&ping)[N], pingattempts &a)
{
    ucharbuf p(ping, N);
    putint(p, a.addattempt(totalmillis));
    buf.data = ping;
    buf.dataLength = p.length();
}

void pingservers()
{
    if(pingsock == ENET_SOCKET_NULL) 
    {
        pingsock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
        if(pingsock == ENET_SOCKET_NULL)
        {
            lastinfo = totalmillis;
            return;
        }
        enet_socket_set_option(pingsock, ENET_SOCKOPT_NONBLOCK, 1);
        enet_socket_set_option(pingsock, ENET_SOCKOPT_BROADCAST, 1);

        lanpings.setoffset();
    }

    ENetBuffer buf;
    uchar ping[MAXTRANS];

    static int lastping = 0;
    if(lastping >= servers.length()) lastping = 0;
    loopi(maxservpings ? min(servers.length(), maxservpings) : servers.length())
    {
        serverinfo &si = *servers[lastping];
        if(++lastping >= servers.length()) lastping = 0;
        if(si.address.host == ENET_HOST_ANY) continue;
        buildping(buf, ping, si);
        enet_socket_send(pingsock, &si.address, &buf, 1);
        
        si.checkdecay(servpingdecay);
    }
    if(searchlan)
    {
        ENetAddress address;
        address.host = ENET_HOST_BROADCAST;
        address.port = server::laninfoport();
        buildping(buf, ping, lanpings);
        enet_socket_send(pingsock, &address, &buf, 1);
    }
    lastinfo = totalmillis;
}
  
void checkresolver()
{
    int resolving = 0;
    loopv(servers)
    {
        serverinfo &si = *servers[i];
        if(si.resolved == RESOLVED) continue;
        if(si.address.host == ENET_HOST_ANY)
        {
            if(si.resolved == UNRESOLVED) { si.resolved = RESOLVING; resolverquery(si.name); }
            resolving++;
        }
    }
    if(!resolving) return;

    const char *name = NULL;
    for(;;)
    {
        ENetAddress addr = { ENET_HOST_ANY, ENET_PORT_ANY };
        if(!resolvercheck(&name, &addr)) break;
        loopv(servers)
        {
            serverinfo &si = *servers[i];
            if(name == si.name)
            {
                si.resolved = RESOLVED; 
                si.address.host = addr.host;
                break;
            }
        }
    }
}

static int lastreset = 0;

void checkpings()
{
    if(pingsock==ENET_SOCKET_NULL) return;
    enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
    ENetBuffer buf;
    ENetAddress addr;
    uchar ping[MAXTRANS];
    char text[MAXTRANS];
    buf.data = ping; 
    buf.dataLength = sizeof(ping);
    while(enet_socket_wait(pingsock, &events, 0) >= 0 && events)
    {
        int len = enet_socket_receive(pingsock, &addr, &buf, 1);
        if(len <= 0) return;  
        ucharbuf p(ping, len);
        int millis = getint(p);
        serverinfo *si = NULL;
        loopv(servers) if(addr.host == servers[i]->address.host && addr.port == servers[i]->address.port) { si = servers[i]; break; }
        if(si)
        {
            if(!si->checkattempt(millis)) continue;
            millis = si->decodeping(millis);
        }
        else if(!searchlan || !lanpings.checkattempt(millis, false)) continue;
        else
        {
            si = newserver(NULL, server::serverport(addr.port), addr.host); 
            millis = lanpings.decodeping(millis);
        }
        int rtt = clamp(totalmillis - millis, 0, min(servpingdecay, totalmillis));
        if(millis >= lastreset && rtt < servpingdecay) si->addping(rtt, millis);
        si->numplayers = getint(p);
        int numattr = getint(p);
        si->attr.setsize(0);
        loopj(numattr) { int attr = getint(p); if(p.overread()) break; si->attr.add(attr); }
        getstring(text, p);
        filtertext(si->map, text, false);
        getstring(text, p);
        filtertext(si->sdesc, text, true, true);
    }
}

void sortservers()
{
    servers.sort(serverinfo::compare);
}
COMMAND(sortservers, "");

VARP(autosortservers, 0, 1, 1);
VARP(autoupdateservers, 0, 1, 1);

#if __EMSCRIPTEN__
void injectserver(char* name, int port, uchar* ping, int numBytes)
{
    serverinfo *si = NULL;
    char text[MAXTRANS];
    ucharbuf p(ping, numBytes);
    //for (int i = 0; i < numBytes; i++) {
        //conoutf("ping[%d]=%d (%c)", i, ping[i], ping[i]);
    //}
    int millis = getint(p);
    loopv(servers)
    {
        if(strcmp(name, servers[i]->name) && port == servers[i]->port) { si = servers[i]; break; }
    }
    if (si == NULL)
    {
        conoutf("Failed to find server matching %s:%d", name, port);
        return;
    }
    si->numplayers = getint(p);
    conoutf("numplayers=%d", si->numplayers);
    int numattr = getint(p);
    conoutf("numattr=%d", numattr);
    si->attr.setsize(0);
    loopj(numattr) { int attr = getint(p); if(p.overread()) break; si->attr.add(attr); conoutf("attr=%d", attr); }
    getstring(text, p);
    conoutf("map=%s", text);
    filtertext(si->map, text, false);
    getstring(text, p);
    conoutf("desc=%s", text);
    filtertext(si->sdesc, text, true, true);
    //conoutf("Data loaded for %s:%d players=%d map=%s desc=%s", name, port, si->numplayers, si->map, si->sdesc);
}
#endif

void refreshservers()
{
#if __EMSCRIPTEN__
    return;
#endif
    static int lastrefresh = 0;
    if(lastrefresh==totalmillis) return;
    if(totalmillis - lastrefresh > 1000) 
    {
        loopv(servers) servers[i]->reset();
        lastreset = totalmillis;
    }
    lastrefresh = totalmillis;

    checkresolver();
    checkpings();
    if(totalmillis - lastinfo >= servpingrate/(maxservpings ? max(1, (servers.length() + maxservpings - 1) / maxservpings) : 1)) pingservers();
    if(autosortservers) sortservers();
}

serverinfo *selectedserver = NULL;

const char *showservers(g3d_gui *cgui, uint *header, int pagemin, int pagemax)
{
    refreshservers();
    if(servers.empty())
    {
        if(header) execute(header);
        return NULL;
    }
    serverinfo *sc = NULL;
    for(int start = 0; start < servers.length();)
    {
        if(start > 0) cgui->tab();
        if(header) execute(header);
        int end = servers.length();
        cgui->pushlist();
        loopi(10)
        {
            if(!game::serverinfostartcolumn(cgui, i)) break;
            for(int j = start; j < end; j++)
            {
                if(!i && j+1 - start >= pagemin && (j+1 - start >= pagemax || cgui->shouldtab())) { end = j; break; }
                serverinfo &si = *servers[j];
                const char *sdesc = si.sdesc;
                if(si.address.host == ENET_HOST_ANY) sdesc = "[unknown host]";
                else if(si.ping == serverinfo::WAITING) sdesc = "[waiting for response]";
                if(game::serverinfoentry(cgui, i, si.name, si.port, sdesc, si.map, sdesc == si.sdesc ? si.ping : -1, si.attr, si.numplayers))
                    sc = &si;
            }
            game::serverinfoendcolumn(cgui, i);
        }
        cgui->poplist();
        start = end;
    }
    if(selectedserver || !sc) return NULL;
    selectedserver = sc;
    return "connectselected";
}

void connectselected()
{
    if(!selectedserver) return;
    connectserv(selectedserver->name, selectedserver->port, selectedserver->password);
    selectedserver = NULL;
}

COMMAND(connectselected, "");

void clearservers(bool full = false)
{
    resolverclear();
    if(full) servers.deletecontents();
    else loopvrev(servers) if(!servers[i]->keep) delete servers.remove(i);
    selectedserver = NULL;
}

#define RETRIEVELIMIT 20000

void retrieveservers(vector<char> &data)
{
    ENetSocket sock = connectmaster(true);
    if(sock == ENET_SOCKET_NULL) return;

    extern char *mastername;
    defformatstring(text, "retrieving servers from %s... (esc to abort)", mastername);
    renderprogress(0, text);

    int starttime = SDL_GetTicks(), timeout = 0;
    const char *req = "list\n";
    int reqlen = strlen(req);
    ENetBuffer buf;
    while(reqlen > 0)
    {
        enet_uint32 events = ENET_SOCKET_WAIT_SEND;
        if(enet_socket_wait(sock, &events, 250) >= 0 && events) 
        {
            buf.data = (void *)req;
            buf.dataLength = reqlen;
            int sent = enet_socket_send(sock, NULL, &buf, 1);
            if(sent < 0) break;
            req += sent;
            reqlen -= sent;
            if(reqlen <= 0) break;
        }
        timeout = SDL_GetTicks() - starttime;
        renderprogress(min(float(timeout)/RETRIEVELIMIT, 1.0f), text);
        if(interceptkey(SDLK_ESCAPE)) timeout = RETRIEVELIMIT + 1;
        if(timeout > RETRIEVELIMIT) break;
    }

    if(reqlen <= 0) for(;;)
    {
        enet_uint32 events = ENET_SOCKET_WAIT_RECEIVE;
        if(enet_socket_wait(sock, &events, 250) >= 0 && events)
        {
            if(data.length() >= data.capacity()) data.reserve(4096);
            buf.data = data.getbuf() + data.length();
            buf.dataLength = data.capacity() - data.length();
            int recv = enet_socket_receive(sock, NULL, &buf, 1);
            if(recv <= 0) break;
            data.advance(recv);
        }
        timeout = SDL_GetTicks() - starttime;
        renderprogress(min(float(timeout)/RETRIEVELIMIT, 1.0f), text);
        if(interceptkey(SDLK_ESCAPE)) timeout = RETRIEVELIMIT + 1;
        if(timeout > RETRIEVELIMIT) break;
    }

    if(data.length()) data.add('\0');
    enet_socket_destroy(sock);
}

bool updatedservers = false;

void updatefrommaster()
{
#if __EMSCRIPTEN__
    // We update this in other ways in Sour
    return;
#endif
    vector<char> data;
    retrieveservers(data);
    if(data.empty()) conoutf(CON_ERROR, "master server not replying");
    else
    {
        clearservers();
        char *line = data.getbuf();
        while(char *end = (char *)memchr(line, '\n', data.length() - (line - data.getbuf())))
        {
            *end = '\0';

            const char *args = line;
            while(args < end && !iscubespace(*args)) args++;
            int cmdlen = args - line;
            while(args < end && iscubespace(*args)) args++;

            if(matchstring(line, cmdlen, "addserver"))
            {
                string ip;
                int port;
                if(sscanf(args, "%100s %d", ip, &port) == 2) addserver(ip, port);
            }
            else if(matchstring(line, cmdlen, "echo")) conoutf("\f1%s", args);

            line = end + 1;
        }
    }
    refreshservers();
    updatedservers = true;
}

void initservers()
{
    selectedserver = NULL;
    if(autoupdateservers && !updatedservers) updatefrommaster();
}

ICOMMAND(addserver, "sis", (const char *name, int *port, const char *password), addserver(name, *port, password[0] ? password : NULL));
ICOMMAND(keepserver, "sis", (const char *name, int *port, const char *password), addserver(name, *port, password[0] ? password : NULL, true));
ICOMMAND(clearservers, "i", (int *full), clearservers(*full!=0));
COMMAND(updatefrommaster, "");
COMMAND(initservers, "");

void writeservercfg()
{
    if(!game::savedservers()) return;
    stream *f = openutf8file(path(game::savedservers(), true), "w");
    if(!f) return;
    int kept = 0;
    loopv(servers)
    {
        serverinfo *s = servers[i];
        if(s->keep)
        {
            if(!kept) f->printf("// servers that should never be cleared from the server list\n\n");
            if(s->password) f->printf("keepserver %s %d %s\n", escapeid(s->name), s->port, escapestring(s->password));
            else f->printf("keepserver %s %d\n", escapeid(s->name), s->port);
            kept++;
        }
    }
    if(kept) f->printf("\n");
    f->printf("// servers connected to are added here automatically\n\n");
    loopv(servers) 
    {
        serverinfo *s = servers[i];
        if(!s->keep) 
        {
            if(s->password) f->printf("addserver %s %d %s\n", escapeid(s->name), s->port, escapestring(s->password));
            else f->printf("addserver %s %d\n", escapeid(s->name), s->port);
        }
    }
    delete f;
}

