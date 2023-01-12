// server.cpp: little more than enhanced multicaster
// runs dedicated or as client coroutine
// includes threading for QServ and IRC, and geoip database initaliziation

#include "../mod/QServ.h"

server::QServ qs(olanguagewarn, maxolangwarnings, commandprefix);

#define LOGSTRLEN 512

static FILE *logfile = NULL;

void closelogfile()
{
    if(logfile)
    {
        fclose(logfile);
        logfile = NULL;
    }
}

FILE *getlogfile()
{
#ifdef WIN32
    return logfile;
#else
    return logfile ? logfile : stdout;
#endif
}

void setlogfile(const char *fname)
{
    closelogfile();
    if(fname && fname[0])
    {
        fname = findfile(fname, "w");
        if(fname) logfile = fopen(fname, "w");
    }
    FILE *f = getlogfile();
    if(f) setvbuf(f, NULL, _IOLBF, BUFSIZ);
}

void logoutf(const char *fmt, ...)
{
    va_list args;
    va_start(args, fmt);
    logoutfv(fmt, args);
    va_end(args);
}

static void writelog(FILE *file, const char *fmt, va_list args)
{
    static char buf[LOGSTRLEN];
    static uchar ubuf[512];
    vformatstring(buf, fmt, args, sizeof(buf));
    int len = strlen(buf), carry = 0;
    while(carry < len)
    {
        int numu = encodeutf8(ubuf, sizeof(ubuf)-1, &((uchar *)buf)[carry], len - carry, &carry);
        if(carry >= len) ubuf[numu++] = '\n';
        fwrite(ubuf, 1, numu, file);
    }
}

void fatal(const char *fmt, ...)
{
    void cleanupserver();
    cleanupserver();
    defvformatstring(msg,fmt,fmt);
    if(logfile) logoutf("%s", msg);
#ifdef WIN32
    MessageBox(NULL, msg, "Cube 2: Sauerbraten fatal error", MB_OK|MB_SYSTEMMODAL);
#else
    fprintf(stderr, "server error: %s\n", msg);
#endif
    closelogfile();
    exit(EXIT_FAILURE);
}

void conoutfv(int type, const char *fmt, va_list args)
{
    string sf, sp;
    vformatstring(sf, fmt, args);
    filtertext(sp, sf);
    logoutf("%s", sp);
}

void conoutf(const char *fmt, ...)
{
    va_list args;
    va_start(args, fmt);
    conoutfv(CON_INFO, fmt, args);
    va_end(args);
}

void conoutf(int type, const char *fmt, ...)
{
    va_list args;
    va_start(args, fmt);
    conoutfv(type, fmt, args);
    va_end(args);
}

// all network traffic is in 32bit ints, which are then compressed using the following simple scheme (assumes that most values are small).

template<class T>
static inline void putint_(T &p, int n)
{
    if(n<128 && n>-127) p.put(n);
    else if(n<0x8000 && n>=-0x8000) { p.put(0x80); p.put(n); p.put(n>>8); }
    else { p.put(0x81); p.put(n); p.put(n>>8); p.put(n>>16); p.put(n>>24); }
}
void putint(ucharbuf &p, int n) { putint_(p, n); }
void putint(packetbuf &p, int n) { putint_(p, n); }
void putint(vector<uchar> &p, int n) { putint_(p, n); }

int getint(ucharbuf &p)
{
    int c = (char)p.get();
    if(c==-128) { int n = p.get(); n |= char(p.get())<<8; return n; }
    else if(c==-127) { int n = p.get(); n |= p.get()<<8; n |= p.get()<<16; return n|(p.get()<<24); }
    else return c;
}

// much smaller encoding for unsigned integers up to 28 bits, but can handle signed
template<class T>
static inline void putuint_(T &p, int n)
{
    if(n < 0 || n >= (1<<21))
    {
        p.put(0x80 | (n & 0x7F));
        p.put(0x80 | ((n >> 7) & 0x7F));
        p.put(0x80 | ((n >> 14) & 0x7F));
        p.put(n >> 21);
    }
    else if(n < (1<<7)) p.put(n);
    else if(n < (1<<14))
    {
        p.put(0x80 | (n & 0x7F));
        p.put(n >> 7);
    }
    else
    {
        p.put(0x80 | (n & 0x7F));
        p.put(0x80 | ((n >> 7) & 0x7F));
        p.put(n >> 14);
    }
}
void putuint(ucharbuf &p, int n) { putuint_(p, n); }
void putuint(packetbuf &p, int n) { putuint_(p, n); }
void putuint(vector<uchar> &p, int n) { putuint_(p, n); }

int getuint(ucharbuf &p)
{
    int n = p.get();
    if(n & 0x80)
    {
        n += (p.get() << 7) - 0x80;
        if(n & (1<<14)) n += (p.get() << 14) - (1<<14);
        if(n & (1<<21)) n += (p.get() << 21) - (1<<21);
        if(n & (1<<28)) n |= -1<<28;
    }
    return n;
}

template<class T>
static inline void putfloat_(T &p, float f)
{
    lilswap(&f, 1);
    p.put((uchar *)&f, sizeof(float));
}
void putfloat(ucharbuf &p, float f) { putfloat_(p, f); }
void putfloat(packetbuf &p, float f) { putfloat_(p, f); }
void putfloat(vector<uchar> &p, float f) { putfloat_(p, f); }

float getfloat(ucharbuf &p)
{
    float f;
    p.get((uchar *)&f, sizeof(float));
    return lilswap(f);
}

template<class T>
static inline void sendstring_(const char *t, T &p)
{
    while(*t) putint(p, *t++);
    putint(p, 0);
}
void sendstring(const char *t, ucharbuf &p) { sendstring_(t, p); }
void sendstring(const char *t, packetbuf &p) { sendstring_(t, p); }
void sendstring(const char *t, vector<uchar> &p) { sendstring_(t, p); }

void getstring(char *text, ucharbuf &p, int len)
{
    char *t = text;
    do
    {
        if(t>=&text[len]) { text[len-1] = 0; return; }
        if(!p.remaining()) { *t = 0; return; }
        *t = getint(p);
    }
    while(*t++);
}

enum { ST_EMPTY, ST_LOCAL, ST_TCPIP, ST_SOCKET };

struct client                   // server side version of "dynent" type
{
    int type;
    int num;
    ushort id; // for socket comms
    ENetPeer *peer;
    string hostname;
    void *info;
};

vector<client *> clients;

ENetHost *serverhost = NULL;
int laststatus = 0;
ENetSocket pongsock = ENET_SOCKET_NULL, lansock = ENET_SOCKET_NULL;

int localclients = 0, nonlocalclients = 0;

bool hasnonlocalclients() { return nonlocalclients!=0; }
bool haslocalclients() { return localclients!=0; }

client &addclient(int type)
{
    client *c = NULL;
    loopv(clients) if(clients[i]->type==ST_EMPTY)
    {
        c = clients[i];
        break;
    }
    if(!c)
    {
        c = new client;
        c->num = clients.length();
        clients.add(c);
    }
    c->info = server::newclientinfo();
    c->type = type;
    switch(type)
    {
        case ST_SOCKET:
        case ST_TCPIP:
            nonlocalclients++; break;
        case ST_LOCAL: localclients++; break;
    }
    return *c;
}

client *findclient(uint id)
{
    client *c = NULL;
    loopv(clients) if(clients[i]->id == id) c = clients[i];
    return c;
}

void delclient(client *c)
{
    if(!c) return;
    switch(c->type)
    {
        case ST_SOCKET:
            c->id = 0;
            nonlocalclients--;
            break;
        case ST_TCPIP:
            nonlocalclients--; if(c->peer) c->peer->data = NULL; break;
        case ST_LOCAL: localclients--; break;
        case ST_EMPTY: return;
    }
    c->type = ST_EMPTY;
    c->peer = NULL;
    if(c->info)
    {
        server::deleteclientinfo(c->info);
        c->info = NULL;
    }
}

void cleanupserver()
{
    if(serverhost) enet_host_destroy(serverhost);
    serverhost = NULL;

    if(pongsock != ENET_SOCKET_NULL) enet_socket_destroy(pongsock);
    if(lansock != ENET_SOCKET_NULL) enet_socket_destroy(lansock);
    pongsock = lansock = ENET_SOCKET_NULL;
}

void process(ENetPacket *packet, int sender, int chan);
//void disconnect_client(int n, int reason);

int getservermtu() { return serverhost ? serverhost->mtu : -1; }
void *getclientinfo(int i) { return !clients.inrange(i) || clients[i]->type==ST_EMPTY ? NULL : clients[i]->info; }
ENetPeer *getclientpeer(int i) { return clients.inrange(i) && clients[i]->type==ST_TCPIP ? clients[i]->peer : NULL; }
int getnumclients()        { return clients.length(); }
uint getclientip(int n)    { return clients.inrange(n) && clients[n]->type==ST_TCPIP ? clients[n]->peer->address.host : 0; }

void sendtocluster(int chan, ENetPacket *packet)
{
    if (socketCtl.isConnected() && clients.length() > 0) {
        // We want the cluster to be able to see all of the broadcasts that
        // happen instead of having to sort through client messages.
        packetbuf p(MAXTRANS);
        putuint(p, SERVER_EVENT_BROADCAST);
        putuint(p, packet->dataLength);
        putuint(p, chan);
        p.put(packet->data, packet->dataLength);
        ENetPacket *newPacket = p.finalize();
        socketCtl.send((char*) newPacket->data, newPacket->dataLength);
    }
}

void sendedit(int client, ENetPacket *packet)
{
    if (socketCtl.isConnected() && clients.length() > 0) {
        packetbuf p(MAXTRANS);
        putuint(p, SERVER_EVENT_EDIT);
        putint(p, client);
        putuint(p, packet->dataLength);
        p.put(packet->data, packet->dataLength);
        ENetPacket *newPacket = p.finalize();
        socketCtl.send((char*) newPacket->data, newPacket->dataLength);
    }
}

void sendpacket(int n, int chan, ENetPacket *packet, int exclude)
{
    #ifdef QDEBUG
    out(ECHO_CONSOLE, "chan: %d, packetData: %s\n\n", chan, packet->data);
    #endif
    if(n<0)
    {
        server::recordpacket(chan, packet->data, packet->dataLength);

        sendtocluster(chan, packet);

        loopv(clients) {
            if(i!=exclude && server::allowbroadcast(i)) sendpacket(i, chan, packet);
        }
        return;
    }
    switch(clients[n]->type)
    {
        case ST_TCPIP:
        {
            enet_peer_send(clients[n]->peer, chan, packet);
            break;
        }
        case ST_SOCKET:
        {
            packetbuf p(MAXTRANS);
            putuint(p, SERVER_EVENT_PACKET);
            putuint(p, packet->dataLength);
            putuint(p, clients[n]->id);
            putuint(p, chan);
            p.put(packet->data, packet->dataLength);
            ENetPacket *newPacket = p.finalize();
            socketCtl.send((char*) newPacket->data, newPacket->dataLength);
            break;
        }
    }
}

ENetPacket *sendf(int cn, int chan, const char *format, ...)
{
    int exclude = -1;
    bool reliable = false;
    if(*format=='r') { reliable = true; ++format; }
    packetbuf p(MAXTRANS, reliable ? ENET_PACKET_FLAG_RELIABLE : 0);
    va_list args;
    va_start(args, format);
    while(*format) switch(*format++)
    {
        case 'x':
            exclude = va_arg(args, int);
            break;

        case 'v':
        {
            int n = va_arg(args, int);
            int *v = va_arg(args, int *);
            loopi(n) putint(p, v[i]);
            break;
        }

        case 'i':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putint(p, va_arg(args, int));
            break;
        }
        case 'f':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putfloat(p, (float)va_arg(args, double));
            break;
        }
        case 's': sendstring(va_arg(args, const char *), p); break;
        case 'm':
        {
            int n = va_arg(args, int);
            p.put(va_arg(args, uchar *), n);
            break;
        }
    }
    va_end(args);
    ENetPacket *packet = p.finalize();
    sendpacket(cn, chan, packet, exclude);
    return packet->referenceCount > 0 ? packet : NULL;
}

void sendsockf(const char *format, ...)
{
    int exclude = -1;
    packetbuf p(MAXTRANS, 0);
    va_list args;
    va_start(args, format);
    while(*format) switch(*format++)
    {
        case 'x':
            exclude = va_arg(args, int);
            break;

        case 'v':
        {
            int n = va_arg(args, int);
            int *v = va_arg(args, int *);
            loopi(n) putint(p, v[i]);
            break;
        }

        case 'i':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putint(p, va_arg(args, int));
            break;
        }

        case 'u':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putuint(p, va_arg(args, int));
            break;
        }

        case 'f':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putfloat(p, (float)va_arg(args, double));
            break;
        }

        case 's': sendstring(va_arg(args, const char *), p); break;

        case 'm':
        {
            int n = va_arg(args, int);
            p.put(va_arg(args, uchar *), n);
            break;
        }
    }
    va_end(args);
    ENetPacket *newPacket = p.finalize();
    socketCtl.send((char*) newPacket->data, newPacket->dataLength);
}

ENetPacket *sendfile(int cn, int chan, stream *file, const char *format, ...)
{
    if(cn < 0)
    {
        return NULL;
    }
    else if(!clients.inrange(cn)) return NULL;

    int len = (int)min(file->size(), stream::offset(INT_MAX));
    if(len <= 0 || len > 16<<20) return NULL;

    packetbuf p(MAXTRANS+len, ENET_PACKET_FLAG_RELIABLE);
    va_list args;
    va_start(args, format);
    while(*format) switch(*format++)
    {
        case 'i':
        {
            int n = isdigit(*format) ? *format++-'0' : 1;
            loopi(n) putint(p, va_arg(args, int));
            break;
        }
        case 's': sendstring(va_arg(args, const char *), p); break;
        case 'l': putint(p, len); break;
    }
    va_end(args);

    file->seek(0, SEEK_SET);
    file->read(p.subbuf(len).buf, len);

    ENetPacket *packet = p.finalize();
    if(cn >= 0) sendpacket(cn, chan, packet, -1);
    return packet->referenceCount > 0 ? packet : NULL;
}

const char *disconnectreason(int reason)
{
    switch(reason)
    {
        case DISC_EOP: return "end of packet";
        case DISC_LOCAL: return "server is in local mode";
        case DISC_KICK: return "kicked/banned";
        case DISC_MSGERR: return "message error";
        case DISC_IPBAN: return "ip is banned";
        case DISC_PRIVATE: return "server is in private mode";
        case DISC_MAXCLIENTS: return "server FULL";
        case DISC_TIMEOUT: return "connection timed out";
        case DISC_OVERFLOW: return "overflow";
        case DISC_PASSWORD: return "invalid password";
        default: return NULL;
    }
}

void disconnect_socket(int n, int reason) {
    if(!clients.inrange(n) ) return;
    const char *msg = disconnectreason(reason);
    sendsockf("uuis", SERVER_EVENT_DISCONNECT, clients[n]->id, reason, msg);
}

void connect_client(int n) {
    if(!clients.inrange(n) ) return;
    sendsockf("uui", SERVER_EVENT_CONNECT, clients[n]->id, n);
}

void requestmap(const char *mapname, int mode) {
    packetbuf p(MAXTRANS);
    sendsockf("usi", SERVER_EVENT_REQUEST_MAP, mapname, mode);
}

void setclientname(int n, const char *name) {
    packetbuf p(MAXTRANS);
    sendsockf("uus", SERVER_EVENT_NAME, clients[n]->id, name);
}

void healthy() {
    packetbuf p(MAXTRANS);
    sendsockf("u", SERVER_EVENT_HEALTHY);
}

VAR(serverdisconnectmsg, 0, 1, 1); //enable/disable msg
void disconnect_client(int n, int reason) {
    qs.resetoLangWarn(n);
    if(!clients.inrange(n) ) return;

    if (clients[n]->type==ST_TCPIP) {
        enet_peer_disconnect(clients[n]->peer, reason);
    }

    const char *msg = disconnectreason(reason);

    if (clients[n]->type==ST_SOCKET) {
        disconnect_socket(n, reason);
    }

    logoutf("Leave: disconnected");
    server::clientdisconnect(n);
    delclient(clients[n]);
    string s;
    if(getvar("ircignore") == 0) serverdisconnectmsg = false; // can cause excess flood
    if(getvar("serverdisconnectmsg")) {
        if(msg) formatstring(s)("client (%s) disconnected because: %s", clients[n]->hostname, msg);
        else formatstring(s)("client (%s) disconnected", clients[n]->hostname);
        server::sendservmsg(s);
    }
    logoutf("%s", s);
}

void dcres(int n, const char *reason) {
    qs.resetoLangWarn(n);
    if(!clients.inrange(n) || clients[n]->type!=ST_TCPIP) return;

    if (clients[n]->type==ST_TCPIP) {
        enet_peer_disconnect(clients[n]->peer, DISC_KICK);
    } else if (clients[n]->type==ST_SOCKET) {
        disconnect_socket(n, DISC_KICK);
    }

    server::clientdisconnect(n);
    delclient(clients[n]);
    string s;
    if(reason) formatstring(s)("client (%s) kicked by server for: \f3%s", clients[n]->hostname, reason);
    else formatstring(s)("client (%s) kicked by server", clients[n]->hostname);
    logoutf("%s", s);
    server::sendservmsg(s);
}

void kicknonlocalclients(int reason)
{
    loopv(clients) if(clients[i]->type==ST_TCPIP) disconnect_client(i, reason);
}

void process(ENetPacket *packet, int sender, int chan)   // sender may be -1
{
    packetbuf p(packet);
    server::parsepacket(sender, chan, p);
    if(p.overread()) { disconnect_client(sender, DISC_EOP); return; }
}

void localclienttoserver(int chan, ENetPacket *packet)
{
    client *c = NULL;
    loopv(clients) if(clients[i]->type==ST_LOCAL) { c = clients[i]; break; }
    if(c) process(packet, c->num, chan);
}

bool resolverwait(const char *name, ENetAddress *address)
{
    return enet_address_set_host(address, name) >= 0;
}

int connectwithtimeout(ENetSocket sock, const char *hostname, const ENetAddress &remoteaddress)
{
    int result = enet_socket_connect(sock, &remoteaddress);
    if(result<0) enet_socket_destroy(sock);
    return result;
}

ENetSocket mastersock = ENET_SOCKET_NULL;
ENetAddress masteraddress = { ENET_HOST_ANY, ENET_PORT_ANY }, serveraddress = { ENET_HOST_ANY, ENET_PORT_ANY };
int lastupdatemaster = 0;
vector<char> masterout, masterin;
int masteroutpos = 0, masterinpos = 0;
VARN(updatemaster, allowupdatemaster, 0, 1, 1);

void disconnectmaster()
{
    if(mastersock != ENET_SOCKET_NULL)
    {
        enet_socket_destroy(mastersock);
        mastersock = ENET_SOCKET_NULL;
    }

    masterout.setsize(0);
    masterin.setsize(0);
    masteroutpos = masterinpos = 0;

    masteraddress.host = ENET_HOST_ANY;
    masteraddress.port = ENET_PORT_ANY;

    lastupdatemaster = 0;
}

SVAR(configpath, "./config/server-init.cfg");
SVARF(mastername, server::defaultmaster(), disconnectmaster());
VARF(masterport, 1, server::masterport(), 0xFFFF, disconnectmaster());

ENetSocket connectmaster()
{
    if(!mastername[0]) return ENET_SOCKET_NULL;

    if(masteraddress.host == ENET_HOST_ANY)
    {
        logoutf("[ OK ] looking up %s...", mastername);
        masteraddress.port = masterport;
        if(!resolverwait(mastername, &masteraddress)) return ENET_SOCKET_NULL;
    }
    ENetSocket sock = enet_socket_create(ENET_SOCKET_TYPE_STREAM);
    if(sock != ENET_SOCKET_NULL && serveraddress.host != ENET_HOST_ANY && enet_socket_bind(sock, &serveraddress) < 0)
    {
        enet_socket_destroy(sock);
        sock = ENET_SOCKET_NULL;
    }
    if(sock == ENET_SOCKET_NULL || connectwithtimeout(sock, mastername, masteraddress) < 0)
    {
        logoutf(sock==ENET_SOCKET_NULL ? "[ FATAL ] could not open socket" : "[ FATAL ] could not connect");
        return ENET_SOCKET_NULL;
    }

    enet_socket_set_option(sock, ENET_SOCKOPT_NONBLOCK, 1);
    return sock;
}

bool requestmaster(const char *req)
{
    if(mastersock == ENET_SOCKET_NULL)
    {
        mastersock = connectmaster();
        if(mastersock == ENET_SOCKET_NULL) return false;
    }

    masterout.put(req, strlen(req));
    return true;
}

bool requestmasterf(const char *fmt, ...)
{
    defvformatstring(req, fmt, fmt);
    return requestmaster(req);
}

void processmasterinput()
{
    if(masterinpos >= masterin.length()) return;

    char *input = &masterin[masterinpos], *end = (char *)memchr(input, '\n', masterin.length() - masterinpos);
    while(end)
    {
        *end++ = '\0';

        const char *args = input;
        while(args < end && !iscubespace(*args)) args++;
        int cmdlen = args - input;
        while(args < end && iscubespace(*args)) args++;

        if(!strncmp(input, "failreg", cmdlen))
            conoutf(CON_ERROR, "master server registration failed: %s", args);
        else if(!strncmp(input, "succreg", cmdlen))
            conoutf("[ OK ] Registered to masterserver");
        else server::processmasterinput(input, cmdlen, args);

        masterinpos = end - masterin.getbuf();
        input = end;
        end = (char *)memchr(input, '\n', masterin.length() - masterinpos);
    }

    if(masterinpos >= masterin.length())
    {
        masterin.setsize(0);
        masterinpos = 0;
    }
}

void flushmasteroutput()
{
    if(masterout.empty()) return;

    ENetBuffer buf;
    buf.data = &masterout[masteroutpos];
    buf.dataLength = masterout.length() - masteroutpos;
    int sent = enet_socket_send(mastersock, NULL, &buf, 1);
    if(sent >= 0)
    {
        masteroutpos += sent;
        if(masteroutpos >= masterout.length())
        {
            masterout.setsize(0);
            masteroutpos = 0;
        }
    }
    else disconnectmaster();
}

void flushmasterinput()
{
    if(masterin.length() >= masterin.capacity())
        masterin.reserve(4096);

    ENetBuffer buf;
    buf.data = masterin.getbuf() + masterin.length();
    buf.dataLength = masterin.capacity() - masterin.length();
    int recv = enet_socket_receive(mastersock, NULL, &buf, 1);
    if(recv > 0)
    {
        masterin.advance(recv);
        processmasterinput();
    }
    else disconnectmaster();
}

static ENetAddress pongaddr;

void sendserverinforeply(ucharbuf &p)
{
    packetbuf r(MAXTRANS);
    putuint(r, SERVER_EVENT_SERVER_INFO_REPLY);
    putuint(r, p.len);
    r.put(p.buf, p.len);
    ENetPacket *newPacket = r.finalize();
    socketCtl.send((char*) newPacket->data, newPacket->dataLength);
}

void checkserversockets()        // reply all server info requests
{
    static ENetSocketSet sockset;
    ENET_SOCKETSET_EMPTY(sockset);
    ENetSocket maxsock = pongsock;
    ENET_SOCKETSET_ADD(sockset, pongsock);
    if(mastersock != ENET_SOCKET_NULL)
    {
        maxsock = max(maxsock, mastersock);
        ENET_SOCKETSET_ADD(sockset, mastersock);
    }
    if(lansock != ENET_SOCKET_NULL)
    {
        maxsock = max(maxsock, lansock);
        ENET_SOCKETSET_ADD(sockset, lansock);
    }
    if(enet_socketset_select(maxsock, &sockset, NULL, 0) <= 0) return;

    ENetBuffer buf;
    uchar pong[MAXTRANS];
    loopi(2)
    {
        ENetSocket sock = i ? lansock : pongsock;
        if(sock == ENET_SOCKET_NULL || !ENET_SOCKETSET_CHECK(sockset, sock)) continue;

        buf.data = pong;
        buf.dataLength = sizeof(pong);
        int len = enet_socket_receive(sock, &pongaddr, &buf, 1);
        if(len < 0) return;
        ucharbuf req(pong, len), p(pong, sizeof(pong));
        p.len += len;
        server::serverinforeply(req, p);
    }

    if(mastersock != ENET_SOCKET_NULL && ENET_SOCKETSET_CHECK(sockset, mastersock)) flushmasterinput();
}

#define DEFAULTCLIENTS 8

VARF(maxclients, 0, DEFAULTCLIENTS, MAXCLIENTS, { if(!maxclients) maxclients = DEFAULTCLIENTS; });
VAR(serveruprate, 0, 0, INT_MAX);
SVAR(serverip, "");
VARF(serverport, 0, server::serverport(), 0xFFFF, { if(!serverport) serverport = server::serverport(); });

int curtime = 0, lastmillis = 0, totalmillis = 0;

void updatemasterserver()
{
    if(mastername[0] && allowupdatemaster) requestmasterf("regserv %d\n", serverport);
    lastupdatemaster = totalmillis ? totalmillis : 1;
}

uint totalsecs = 0;

void updatetime()
{
    static int lastsec = 0;
    if(totalmillis - lastsec >= 1000)
    {
        int cursecs = (totalmillis - lastsec) / 1000;
        totalsecs += cursecs;
        lastsec += cursecs * 1000;
    }
}

void serverslice(bool dedicated, bool enet, uint timeout)   // main server update, called from main loop in sp, or from below in dedicated server
{
    if(!serverhost && !dedicated && !enet)
    {
        server::serverupdate();
        server::sendpackets();
        return;
    }

    // below is network only

    if(dedicated)
    {
        int millis = (int)enet_time_get(), elapsed = millis - totalmillis;
        static int timeerr = 0;
        int scaledtime = server::scaletime(elapsed) + timeerr;
        curtime = scaledtime/100;
        timeerr = scaledtime%100;
        if(server::ispaused()) curtime = 0;
        lastmillis += curtime;
        totalmillis = millis;
        updatetime();
    }
    server::serverupdate();

    flushmasteroutput();
    checkserversockets();

    if(totalmillis-laststatus>60*1000 && serverhost)   // display bandwidth stats, useful for server ops
    {
        laststatus = totalmillis;
        if(nonlocalclients || serverhost->totalSentData || serverhost->totalReceivedData) {
            out(ECHO_NOCOLOR,"[ STATUS ] %d remote client(s), %.1f sent, %.1f rec (K/sec)", nonlocalclients, serverhost->totalSentData/60.0f/1024, serverhost->totalReceivedData/60.0f/1024);
        }
        serverhost->totalSentData = serverhost->totalReceivedData = 0;
    }

    // First process socket traffic
    ENetPacket socketRead, message;
    if (socketCtl.receive(&socketRead) != -1) {
        packetbuf p(&socketRead);

        while (!p.overread() && p.len != p.maxlen) {
            uint messageBytes = getuint(p);
            ucharbuf q(p.buf + p.len, messageBytes);
            for (int i = 0; i < messageBytes; i++) p.get();

            uint type = getuint(q);
            switch(type)
            {
                case SOCKET_EVENT_CONNECT:
                    {
                        uint id = getuint(q);
                        client &c = addclient(ST_SOCKET);
                        c.id = id;

                        server::clientinfo * info = (server::clientinfo*) getclientinfo(c.num);
                        info->sourtype = getuint(q);

                        copystring(c.hostname, "unknown");
                        logoutf("Join: (socket:%d)", c.id);
                        int reason = server::clientconnect(c.num, 0, c.hostname);
                        if(reason) disconnect_client(c.num, reason);
                        break;
                    }
                case SOCKET_EVENT_RECEIVE:
                    {
                        uint id = getuint(q);
                        uint channel = getuint(q);
                        client *c = findclient(id);
                        if(!c) break;

                        packetbuf r(MAXTRANS);
                        r.put(q.buf + q.len, messageBytes - q.len);
                        ENetPacket *newPacket = r.finalize();
                        process(newPacket, c->num, channel);
                        break;
                    }
                case SOCKET_EVENT_DISCONNECT:
                    {
                        uint id = getuint(q);
                        client *c = findclient(id);
                        if(!c) break;
                        logoutf("Leave: (socket:%d)", c->id);
                        server::clientdisconnect(c->num);
                        delclient(c);
                        break;
                    }
                case SOCKET_EVENT_COMMAND:
                    {
                        string command;
                        getstring(command, q, sizeof(command));
                        int result = execute(command);
                        //logoutf("ran '%s' result %d", command, result);
                        break;
                    }
                case SOCKET_EVENT_RESPOND_MAP:
                    {
                        string mapName;
                        getstring(mapName, q, sizeof(mapName));
                        int mode = getint(q), succeeded = getint(q);
                        game::_changemap(mapName, mode);
                        break;
                    }
                case SOCKET_EVENT_PING:
                    {
                        sendsockf("u", SERVER_EVENT_PONG);
                        break;
                    }
                case SOCKET_EVENT_SERVER_INFO_REQUEST:
                    {
                        uchar pong[MAXTRANS];
                        ucharbuf res(pong, MAXTRANS);
                        res.put(q.buf + q.len, messageBytes - q.len);
                        server::serverinforeply(q, res);
                        break;
                    }

                default:
                    break;
            }
        }
    }

    if (!enet) {
        server::sendpackets();
        return;
    }

    // Then check enet traffic
    ENetEvent event;
    bool serviced = false;
    while(!serviced)
    {
        if(enet_host_check_events(serverhost, &event) <= 0)
        {
            if(enet_host_service(serverhost, &event, timeout) <= 0) break;
            serviced = true;
        }
        switch(event.type)
        {
            case ENET_EVENT_TYPE_CONNECT:
            {
                client &c = addclient(ST_TCPIP);
                c.peer = event.peer;
                c.peer->data = &c;
                char hn[1024];
                copystring(c.hostname, (enet_address_get_host_ip(&c.peer->address, hn, sizeof(hn))==0) ? hn : "unknown");
                logoutf("Join: (%s)", c.hostname);
                out(ECHO_IRC, "Join: (%s)", c.hostname);
                int reason = server::clientconnect(c.num, c.peer->address.host, c.hostname); //ipstring for QServ
                if(reason) disconnect_client(c.num, reason);
                break;
            }
            case ENET_EVENT_TYPE_RECEIVE:
            {
                client *c = (client *)event.peer->data;
                if(c) process(event.packet, c->num, event.channelID);
                if(event.packet->referenceCount==0) enet_packet_destroy(event.packet);
                break;
            }
            case ENET_EVENT_TYPE_DISCONNECT:
            {
                client *c = (client *)event.peer->data;
                if(!c) break;
                logoutf("Leave: (%s)", c->hostname);
                server::clientdisconnect(c->num);
                delclient(c);
                break;
            }
            default:
                break;
        }
    }
    if(server::sendpackets()) enet_host_flush(serverhost);
}

void flushserver(bool force)
{
    if(server::sendpackets(force) && serverhost) enet_host_flush(serverhost);
}

#ifdef WIN32
#include "shellapi.h"

#define IDI_ICON1 1

static string apptip = "";
static HINSTANCE appinstance = NULL;
static ATOM wndclass = 0;
static HWND appwindow = NULL, conwindow = NULL;
static HICON appicon = NULL;
static HMENU appmenu = NULL;
static HANDLE outhandle = NULL;
static const int MAXLOGLINES = 200;
struct logline { int len; char buf[LOGSTRLEN]; };
static ringbuf<logline, MAXLOGLINES> loglines;

static void cleanupsystemtray()
{
    NOTIFYICONDATA nid;
    memset(&nid, 0, sizeof(nid));
    nid.cbSize = sizeof(nid);
    nid.hWnd = appwindow;
    nid.uID = IDI_ICON1;
    Shell_NotifyIcon(NIM_DELETE, &nid);
}

static bool setupsystemtray(UINT uCallbackMessage)
{
    NOTIFYICONDATA nid;
    memset(&nid, 0, sizeof(nid));
    nid.cbSize = sizeof(nid);
    nid.hWnd = appwindow;
    nid.uID = IDI_ICON1;
    nid.uCallbackMessage = uCallbackMessage;
    nid.uFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP;
    nid.hIcon = appicon;
    strcpy(nid.szTip, apptip);
    if(Shell_NotifyIcon(NIM_ADD, &nid) != TRUE)
        return false;
    atexit(cleanupsystemtray);
    return true;
}

#if 0
static bool modifysystemtray()
{
    NOTIFYICONDATA nid;
    memset(&nid, 0, sizeof(nid));
    nid.cbSize = sizeof(nid);
    nid.hWnd = appwindow;
    nid.uID = IDI_ICON1;
    nid.uFlags = NIF_TIP;
    strcpy(nid.szTip, apptip);
    return Shell_NotifyIcon(NIM_MODIFY, &nid) == TRUE;
}
#endif

static void cleanupwindow()
{
    if(!appinstance) return;
    if(appmenu)
    {
        DestroyMenu(appmenu);
        appmenu = NULL;
    }
    if(wndclass)
    {
        UnregisterClass(MAKEINTATOM(wndclass), appinstance);
        wndclass = 0;
    }
}

static BOOL WINAPI consolehandler(DWORD dwCtrlType)
{
    switch(dwCtrlType)
    {
        case CTRL_C_EVENT:
        case CTRL_BREAK_EVENT:
        case CTRL_CLOSE_EVENT:
            exit(EXIT_SUCCESS);
            return TRUE;
    }
    return FALSE;
}

static void writeline(logline &line)
{
    static uchar ubuf[512];
    int len = strlen(line.buf), carry = 0;
    while(carry < len)
    {
        int numu = encodeutf8(ubuf, sizeof(ubuf), &((uchar *)line.buf)[carry], len - carry, &carry);
        DWORD written = 0;
        WriteConsole(outhandle, ubuf, numu, &written, NULL);
    }
}

static void setupconsole()
{
    if(conwindow) return;
    if(!AllocConsole()) return;
    SetConsoleCtrlHandler(consolehandler, TRUE);
    conwindow = GetConsoleWindow();
    SetConsoleTitle(apptip);
    SendMessage(conwindow, WM_SETICON, ICON_SMALL, (LPARAM)appicon);
    SendMessage(conwindow, WM_SETICON, ICON_BIG, (LPARAM)appicon);
    outhandle = GetStdHandle(STD_OUTPUT_HANDLE);
    CONSOLE_SCREEN_BUFFER_INFO coninfo;
    GetConsoleScreenBufferInfo(outhandle, &coninfo);
    coninfo.dwSize.Y = MAXLOGLINES;
    SetConsoleScreenBufferSize(outhandle, coninfo.dwSize);
    SetConsoleCP(CP_UTF8);
    SetConsoleOutputCP(CP_UTF8);
    loopv(loglines) writeline(loglines[i]);
}

enum
{
    MENU_OPENCONSOLE = 0,
    MENU_SHOWCONSOLE,
    MENU_HIDECONSOLE,
    MENU_EXIT
};

static LRESULT CALLBACK handlemessages(HWND hWnd, UINT uMsg, WPARAM wParam, LPARAM lParam)
{
    switch(uMsg)
    {
        case WM_APP:
            SetForegroundWindow(hWnd);
            switch(lParam)
        {
            case WM_MOUSEMOVE:
                break;
            case WM_LBUTTONUP:
            case WM_RBUTTONUP:
            {
                POINT pos;
                GetCursorPos(&pos);
                TrackPopupMenu(appmenu, TPM_CENTERALIGN|TPM_BOTTOMALIGN|TPM_RIGHTBUTTON, pos.x, pos.y, 0, hWnd, NULL);
                PostMessage(hWnd, WM_NULL, 0, 0);
                break;
            }
        }
            return 0;
        case WM_COMMAND:
            switch(LOWORD(wParam))
        {
            case MENU_OPENCONSOLE:
                setupconsole();
                if(conwindow) ModifyMenu(appmenu, 0, MF_BYPOSITION|MF_STRING, MENU_HIDECONSOLE, "Hide Console");
                break;
            case MENU_SHOWCONSOLE:
                ShowWindow(conwindow, SW_SHOWNORMAL);
                ModifyMenu(appmenu, 0, MF_BYPOSITION|MF_STRING, MENU_HIDECONSOLE, "Hide Console");
                break;
            case MENU_HIDECONSOLE:
                ShowWindow(conwindow, SW_HIDE);
                ModifyMenu(appmenu, 0, MF_BYPOSITION|MF_STRING, MENU_SHOWCONSOLE, "Show Console");
                break;
            case MENU_EXIT:
                PostMessage(hWnd, WM_CLOSE, 0, 0);
                break;
        }
            return 0;
        case WM_CLOSE:
            PostQuitMessage(0);
            return 0;
    }
    return DefWindowProc(hWnd, uMsg, wParam, lParam);
}

static void setupwindow(const char *title, const char *path)
{
    copystring(apptip, title);

    appinstance = GetModuleHandle(path);
    if(!appinstance) fatal("failed getting application instance");
    appicon = LoadIcon(appinstance, MAKEINTRESOURCE(IDI_ICON1));
    (HICON)LoadImage(appinstance, MAKEINTRESOURCE(IDI_ICON1), IMAGE_ICON, 0, 0, LR_DEFAULTSIZE);
    if(!appicon) fatal("failed loading icon");

    appmenu = CreatePopupMenu();
    if(!appmenu) fatal("failed creating popup menu");
    AppendMenu(appmenu, MF_STRING, MENU_OPENCONSOLE, "Open Console");
    AppendMenu(appmenu, MF_SEPARATOR, 0, NULL);
    AppendMenu(appmenu, MF_STRING, MENU_EXIT, "Exit");
    SetMenuDefaultItem(appmenu, 0, FALSE);

    WNDCLASS wc;
    memset(&wc, 0, sizeof(wc));
    wc.hCursor = NULL;
    LoadCursor(NULL, IDC_ARROW);

    wc.hIcon = appicon;
    wc.hIcon = LoadIcon(0, IDI_EXCLAMATION);
    wc.lpszMenuName = NULL;
    wc.lpszClassName = title;
    wc.style = 0;
    wc.hInstance = appinstance;
    wc.lpfnWndProc = handlemessages;
    wc.cbWndExtra = 0;
    wc.cbClsExtra = 0;
    wndclass = RegisterClass(&wc);
    if(!wndclass) fatal("failed registering window class");

    appwindow = CreateWindow(MAKEINTATOM(wndclass), title, 0, CW_USEDEFAULT, CW_USEDEFAULT, 0, 0, HWND_MESSAGE, NULL, appinstance, NULL);
    if(!appwindow) fatal("failed creating window");

    atexit(cleanupwindow);

    if(!setupsystemtray(WM_APP)) fatal("failed adding to system tray");
}

static char *parsecommandline(const char *src, vector<char *> &args)
{
    char *buf = new char[strlen(src) + 1], *dst = buf;
    for(;;)
    {
        while(isspace(*src)) src++;
        if(!*src) break;
        args.add(dst);
        for(bool quoted = false; *src && (quoted || !isspace(*src)); src++)
        {
            if(*src != '"') *dst++ = *src;
            else if(dst > buf && src[-1] == '\\') dst[-1] = '"';
            else quoted = !quoted;
        }
        *dst++ = '\0';
    }
    args.add(NULL);
    return buf;
}


int WINAPI WinMain(HINSTANCE hInst, HINSTANCE hPrev, LPSTR szCmdLine, int sw)
{
    vector<char *> args;
    char *buf = parsecommandline(GetCommandLine(), args);
    appinstance = hInst;
    delete[] buf;
    exit(0);
    return 0;
}

void logoutfv(const char *fmt, va_list args)
{
    if(logfile) writelog(logfile, fmt, args);
    if(appwindow)
    {
        logline &line = loglines.add();
        vformatstring(line.buf, fmt, args, sizeof(line.buf));
        line.len = min(strlen(line.buf), sizeof(line.buf)-2);
        line.buf[line.len++] = '\n';
        line.buf[line.len] = '\0';
        if(outhandle) writeline(line);
    }
}

#else

void logoutfv(const char *fmt, va_list args)
{
    FILE *f = getlogfile();
    if(f) writelog(f, fmt, args);
}

#endif

static bool dedicatedserver = true;

bool isdedicatedserver() { return dedicatedserver; }


pthread_t thread2;

void *main_thread(void*t) {
    for(;;) {
        serverslice(true, false, 5);
    }
    pthread_exit((void*)t);
}

void *main_thread_s(void *t) {
    for(;;)
    {
#ifdef WIN32
        MSG msg;
        while(PeekMessage(&msg, NULL, 0, 0, PM_REMOVE))
        {
            if(msg.message == WM_QUIT) exit(EXIT_SUCCESS);
            TranslateMessage(&msg);
            DispatchMessage(&msg);
        }
#endif
        serverslice(true, false, 5);

    }
    pthread_exit((void*)t);
}

void rundedicatedserver()
{
#ifdef WIN32
    SetPriorityClass(GetCurrentProcess(), HIGH_PRIORITY_CLASS);

#else
#endif
    logoutf("[ OK ] QServ Started, waiting for clients...");
}

bool servererror(bool dedicated, const char *desc)
{
    fatal(desc);
    return false;
}

bool setuplistenserver(bool dedicated)
{
    ENetAddress address = { ENET_HOST_ANY, enet_uint16(serverport <= 0 ? server::serverport() : serverport) };
    if(*serverip)
    {
        if(enet_address_set_host(&address, serverip)<0) conoutf(CON_WARN, "WARNING: server ip not resolved");
        else serveraddress.host = address.host;
    }
    serverhost = enet_host_create(&address, min(maxclients + server::reserveclients(), MAXCLIENTS), server::numchannels(), 0, serveruprate);
    if(!serverhost) return servererror(dedicated, "could not create server host");
    loopi(maxclients) serverhost->peers[i].data = NULL;
    address.port = server::serverinfoport(serverport > 0 ? serverport : -1);

    pongsock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
    if(pongsock != ENET_SOCKET_NULL && enet_socket_bind(pongsock, &address) < 0)
    {
        enet_socket_destroy(pongsock);
        pongsock = ENET_SOCKET_NULL;
    }
    if(pongsock == ENET_SOCKET_NULL) return servererror(dedicated, "could not create server info socket");
    else enet_socket_set_option(pongsock, ENET_SOCKOPT_NONBLOCK, 1);
    address.port = server::laninfoport();
    lansock = enet_socket_create(ENET_SOCKET_TYPE_DATAGRAM);
    if(lansock != ENET_SOCKET_NULL && (enet_socket_set_option(lansock, ENET_SOCKOPT_REUSEADDR, 1) < 0 || enet_socket_bind(lansock, &address) < 0))
    {
        enet_socket_destroy(lansock);
        lansock = ENET_SOCKET_NULL;
    }
    if(lansock == ENET_SOCKET_NULL) conoutf(CON_WARN, "WARNING: could not create LAN server info socket");
    else enet_socket_set_option(lansock, ENET_SOCKOPT_NONBLOCK, 1);
    return true;
}

void initserver(bool listen, bool enet, bool dedicated) //, const char *path
{
    if(dedicated)
    {
        #ifdef WIN32
               setupwindow("QServ", path);
        #endif
    }

    execfile(configpath, false);

    if(enet) setuplistenserver(dedicated);

    server::serverinit();
	logoutf("Protocol version: %d", PROTOCOL_VERSION);

    if(enet)
    {
        updatemasterserver();
        if(dedicated) rundedicatedserver(); // never returns
    }
}

bool serveroption(char *opt)
{
    switch(opt[1])
    {
        case 'u': setvar("serveruprate", atoi(opt+2)); return true;
        case 'S': setsvar("socketpath", opt+2); return true;
        case 'C': setsvar("configpath", opt+2); return true;
        case 'c': setvar("maxclients", atoi(opt+2)); return true;
        case 'i': setsvar("serverip", opt+2); return true;
        case 'j': setvar("serverport", atoi(opt+2)); return true;
        case 'm': setsvar("mastername", opt+2); setvar("updatemaster", mastername[0] ? 1 : 0); return true;
        case 'q': logoutf("Using home directory: %s", opt); sethomedir(opt+2); return true;
        case 'k': logoutf("Adding package directory: %s", opt); addpackagedir(opt+2); return true;
        case 'g': logoutf("Setting log file: %s", opt); setlogfile(opt+2); return true;
        default: return false;
    }
}

vector<const char *> gameargs;

#include "../mod/QCom.h"

int main(int argc, char **argv) {
    srand (time(NULL));
    qs.initCommands(server::initCmds);
    setlogfile(NULL);
    if(enet_initialize()<0) fatal("[FATAL ERROR]: Unable to initialise network module");
    atexit(enet_deinitialize);
    enet_time_set(0);
    for(int i = 1; i<argc; i++) if(argv[i][0]!='-' || !serveroption(argv[i])) gameargs.add(argv[i]);
    game::parseoptions(gameargs);

    socketCtl.init();

    //main server init
    initserver(true, false, true);

    pthread_t thread[2];
    int c; long t;
    pthread_attr_t attr;
    void *status;

    pthread_attr_init(&attr);
    pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_JOINABLE);

  	#ifdef _WIN32
    	c = pthread_create(&thread[0], &attr, main_thread_s, (void*)&t);
    #else
    	c = pthread_create(&thread[0], &attr, main_thread, (void*)&t);
    #endif

    pthread_attr_destroy(&attr);
	for(int i = 0; i < 1; i++) {
		c = pthread_join(thread[i], &status);
        qsleep(5);
    }

    server::serverclose();
    socketCtl.finish();
    //pthread_exit(NULL); //we don't close our thread
    //return EXIT_SUCCESS; //we don't exit
    return 0; //instead, we return with no problems
}
