// client.cpp, mostly network related client game code

#include "engine.h"
#include <emscripten.h>
#include <string.h>

ENetHost *clienthost = NULL;
ENetPeer *curpeer = NULL, *connpeer = NULL;
int connmillis = 0, connattempts = 0, discmillis = 0;

#if __EMSCRIPTEN__
bool sourconnected = false, sourconnecting = false;
#endif

bool multiplayer(bool msg)
{
#if __EMSCRIPTEN__
    bool val = curpeer || sourconnected || hasnonlocalclients(); 
#else
    bool val = curpeer || hasnonlocalclients(); 
#endif
    if(val && msg) conoutf(CON_ERROR, "operation not available in multiplayer");
    return val;
}

void setrate(int rate)
{
   if(!curpeer) return;
   enet_host_bandwidth_limit(clienthost, rate*1024, rate*1024);
}

VARF(rate, 0, 0, 1024, setrate(rate));

void throttle();

VARF(throttle_interval, 0, 5, 30, throttle());
VARF(throttle_accel,    0, 2, 32, throttle());
VARF(throttle_decel,    0, 2, 32, throttle());

void throttle()
{
    if(!curpeer) return;
    ASSERT(ENET_PEER_PACKET_THROTTLE_SCALE==32);
    enet_peer_throttle_configure(curpeer, throttle_interval*1000, throttle_accel, throttle_decel);
}

bool isconnected(bool attempt, bool local)
{
#if __EMSCRIPTEN__
    return curpeer || sourconnected || (attempt && connpeer) || (local && haslocalclients());
#else
    return curpeer || (attempt && connpeer) || (local && haslocalclients());
#endif
}

ICOMMAND(isconnected, "bb", (int *attempt, int *local), intret(isconnected(*attempt > 0, *local != 0) ? 1 : 0));

#if __EMSCRIPTEN__
bool issourserver()
{
    return isconnected(false, false) && sourconnected;
}
ICOMMAND(issourserver, "", (), intret(issourserver() ? 1 : 0));
#endif

const ENetAddress *connectedpeer()
{
    return curpeer ? &curpeer->address : NULL;
}

ICOMMAND(connectedip, "", (),
{
    const ENetAddress *address = connectedpeer();
    string hostname;
    result(address && enet_address_get_host_ip(address, hostname, sizeof(hostname)) >= 0 ? hostname : "");
});

ICOMMAND(connectedport, "", (),
{
    const ENetAddress *address = connectedpeer();
    intret(address ? address->port : -1);
});

void abortconnect()
{
    if(!connpeer) return;
    game::connectfail();
    if(connpeer->state!=ENET_PEER_STATE_DISCONNECTED) enet_peer_reset(connpeer);
    connpeer = NULL;
    if(curpeer) return;
    enet_host_destroy(clienthost);
    clienthost = NULL;
}

SVARP(connectname, "");
VARP(connectport, 0, 0, 0xFFFF);

#if __EMSCRIPTEN__
void abortjoin()
{
    if(!sourconnecting) return;
    EM_ASM({
        Module.cluster.disconnect();
    });
    sourconnected = false;
    sourconnecting = false;
}

void leave(bool async, bool cleanup)
{
    if(!sourconnected) return;
    EM_ASM({
        Module.cluster.disconnect();
    });
    sourconnected = false;
    discmillis = 0;
    conoutf("left");
#if __EMSCRIPTEN__
    EM_ASM({
        Module.onDisconnect();
    });
#endif
    game::gamedisconnect(cleanup);
    mainmenu = 1;
}

void createsourgame(const char *presetname)
{
    EM_ASM({
            Module.cluster.createGame(UTF8ToString($0))
    }, presetname);
}
ICOMMAND(creategame, "s", (char *presetname), createsourgame(presetname));

void tryleave(bool local)
{
    if(sourconnecting)
    {
        conoutf("aborting connection attempt");
        abortjoin();
    }
    else if(sourconnected)
    {
        conoutf("attempting to leave...");
        leave(!discmillis, true);
    }
    else conoutf(CON_WARN, "not connected");
}
ICOMMAND(leave, "b", (int *local), tryleave(*local != 0));

// We don't need to use enet to join Sour servers.
void connectsour(const char *servername, const char *serverpassword)
{   
    abortconnect();
    if(curpeer)
    {
        disconnect(!discmillis);
    }
    if(sourconnected)
    {
        leave(!discmillis, false);
    }

    if(sourconnecting)
    {
        conoutf("aborting connection attempt");
        abortjoin();
    }

    connmillis = totalmillis;
    connattempts = 0;
    sourconnecting = true;

    if(strcmp(servername, connectname)) setsvar("connectname", servername);
    setvar("connectport", 0);

    EM_ASM({
            Module.cluster.connect(UTF8ToString($0), UTF8ToString($1))
    }, servername, serverpassword);
}
ICOMMAND(join, "ss", (char *name, char *pw), connectsour(name, pw));
#endif

void connectserv(const char *servername, int serverport, const char *serverpassword)
{   
    if(connpeer)
    {
        conoutf("aborting connection attempt");
        abortconnect();
    }

    if(serverport <= 0) serverport = server::serverport();

    ENetAddress address;
    address.port = serverport;

    if(servername)
    {
#if !__EMSCRIPTEN__
        if(strcmp(servername, connectname)) setsvar("connectname", servername);
        if(serverport != connectport) setvar("connectport", serverport);
        conoutf("attempting to connect to %s:%d", servername, serverport);
        addserver(servername, serverport, serverpassword && serverpassword[0] ? serverpassword : NULL);
        if(!resolverwait(servername, &address))
        {
            conoutf(CON_ERROR, "\f3could not resolve server %s", servername);
            return;
        }
#else
        int length = strlen(servername);
        if (length == 0) {
            // Just connect to the default
            connectsour("", "");
            return;
        } else {
            if(strcmp(servername, connectname)) setsvar("connectname", servername);
            if(serverport != connectport) setvar("connectport", serverport);
            enet_address_set_host(&address, servername);
            conoutf("attempting to connect to %s:%d", servername, serverport);
        }
#endif

    }
    else
    {
        setsvar("connectname", "");
        setvar("connectport", 0);
        conoutf("attempting to connect over LAN");
        address.host = ENET_HOST_BROADCAST;
    }

    if(!clienthost) 
    {
        clienthost = enet_host_create(NULL, 2, server::numchannels(), rate*1024, rate*1024);
        if(!clienthost)
        {
            conoutf(CON_ERROR, "\f3could not connect to server");
            return;
        }
        clienthost->duplicatePeers = 0;
    }

    connpeer = enet_host_connect(clienthost, &address, server::numchannels(), 0); 
    enet_host_flush(clienthost);
    connmillis = totalmillis;
    connattempts = 0;

    game::connectattempt(servername ? servername : "", serverpassword ? serverpassword : "", address);
}

void reconnect(const char *serverpassword)
{
    if(!connectname[0] || connectport <= 0)
    {
        conoutf(CON_ERROR, "no previous connection");
        return;
    }

    connectserv(connectname, connectport, serverpassword);
}

void disconnect(bool async, bool cleanup)
{
    if(curpeer) 
    {
        if(!discmillis)
        {
            enet_peer_disconnect(curpeer, DISC_NONE);
            enet_host_flush(clienthost);
            discmillis = totalmillis;
        }
        if(curpeer->state!=ENET_PEER_STATE_DISCONNECTED)
        {
            if(async) return;
            enet_peer_reset(curpeer);
        }
        curpeer = NULL;
        discmillis = 0;
        conoutf("disconnected");
#if __EMSCRIPTEN__
        EM_ASM({
            Module.onDisconnect();
        });
#endif
        game::gamedisconnect(cleanup);
        mainmenu = 1;
    }
    if(!connpeer && clienthost)
    {
        enet_host_destroy(clienthost);
        clienthost = NULL;
    }
	extern bool loading_map_file;
	loading_map_file = false;
}

void trydisconnect(bool local)
{
#if __EMSCRIPTEN__
    if (sourconnected || sourconnecting) {
        tryleave(local);
        return;
    }
#endif
    if(connpeer)
    {
        conoutf("aborting connection attempt");
        abortconnect();
    }
    else if(curpeer)
    {
        conoutf("attempting to disconnect...");
        disconnect(!discmillis);
    }
    else if(local && haslocalclients()) localdisconnect();
    else conoutf(CON_WARN, "not connected");
}

ICOMMAND(connect, "sis", (char *name, int *port, char *pw), connectserv(name, *port, pw));
ICOMMAND(lanconnect, "is", (int *port, char *pw), connectserv(NULL, *port, pw));
COMMAND(reconnect, "s");
ICOMMAND(disconnect, "b", (int *local), trydisconnect(*local != 0));
ICOMMAND(localconnect, "", (), { if(!isconnected()) localconnect(); });
ICOMMAND(localdisconnect, "", (), { if(haslocalclients()) localdisconnect(); });

void sendclientpacket(ENetPacket *packet, int chan)
{
    if(curpeer) enet_peer_send(curpeer, chan, packet);
#if __EMSCRIPTEN__
    else if (sourconnected) {
        EM_ASM({
                Module.cluster.send($0, $1, $2)
        }, chan, packet->data, packet->dataLength);
    }
#endif
    else localclienttoserver(chan, packet);
}

void flushclient()
{
    if(clienthost) enet_host_flush(clienthost);
}

void neterr(const char *s, bool disc)
{
    conoutf(CON_ERROR, "\f3illegal network message (%s)", s);
    if(disc) disconnect();
}

void localservertoclient(int chan, ENetPacket *packet)   // processes any updates from the server
{
    packetbuf p(packet);
    game::parsepacketclient(chan, p);
}

void clientkeepalive() { if(clienthost) enet_host_service(clienthost, NULL, 0); }

void gets2c()           // get updates from the server
{
    ENetEvent event;

#if __EMSCRIPTEN__
    if(totalmillis/3000 > connmillis/3000 && sourconnecting)
    {

        conoutf("attempting to connect...");
        connmillis = totalmillis;
        ++connattempts; 
        if(connattempts > 3)
        {
            conoutf(CON_ERROR, "\f3could not connect to server");
            if (connpeer) {
                abortconnect();
            } else if (sourconnecting) {
                abortjoin();
            }
            return;
        }
    }
#else
    if(!clienthost) return;
    if(connpeer && totalmillis/3000 > connmillis/3000)
    {

        conoutf("attempting to connect...");
        connmillis = totalmillis;
        ++connattempts; 
        if(connattempts > 3)
        {
            conoutf(CON_ERROR, "\f3could not connect to server");
            abortconnect();
            return;
        }
    }
#endif

#if __EMSCRIPTEN__
    ENetPacket packet;
    ushort sourEvent, sourChannel, reason;
    while (true) {
        enet_uint8 * frame = (enet_uint8*) EM_ASM_INT({
            return Module.cluster.receive($0, $1)
        }, &packet.data, &packet.dataLength);

        if (frame == NULL) {
            break;
        }

        sourEvent = *((ushort*) frame);

        if ((int) sourEvent == ENET_EVENT_TYPE_CONNECT) {
            conoutf("joined server");

            EM_ASM({
                Module.onConnect(UTF8ToString($0), $1);
            }, connectname, connectport);

            EM_ASM({
                Module.assets.onConnect();
            });

            sourconnected = true;
            sourconnecting = false;
            game::gameconnect(true);
            break;
        } else if ((int) sourEvent == ENET_EVENT_TYPE_DISCONNECT) {
            reason = *((ushort*) (frame + 2));

            if(sourconnecting)
            {
                conoutf(CON_ERROR, "\f3could not join server");
                abortjoin();
            }
            else
            {
                if(!discmillis)
                {
                    const char *msg = disconnectreason(reason);
                    if(msg) conoutf(CON_ERROR, "\f3server network error, leaving (%s) ...", msg);
                    else conoutf(CON_ERROR, "\f3server network error, leaving...");
                }
                leave(false, false);
            }
            break;
        }

        sourChannel = *((ushort*) (frame + 2));
        packet.flags = 0;
        packet.dataLength = *((size_t*)(frame + 4));
        packet.data = frame + 8;

        if(discmillis) conoutf("attempting to leave...");
        else localservertoclient(sourChannel, &packet);
    }

    if(!clienthost) return;
#endif

    while(clienthost && enet_host_service(clienthost, &event, 0)>0)
    switch(event.type)
    {
        case ENET_EVENT_TYPE_CONNECT:
            disconnect(false, false); 
            localdisconnect(false);
            curpeer = connpeer;
            connpeer = NULL;
            conoutf("connected to server");
#if __EMSCRIPTEN__
            EM_ASM({
                Module.onConnect(UTF8ToString($0), $1);
            }, connectname, connectport);

            EM_ASM({
                Module.assets.onConnect();
            });
#endif
            throttle();
            if(rate) setrate(rate);
            game::gameconnect(true);
            break;
         
        case ENET_EVENT_TYPE_RECEIVE:
            if(discmillis) conoutf("attempting to disconnect...");
            else localservertoclient(event.channelID, event.packet);
            enet_packet_destroy(event.packet);
            break;

        case ENET_EVENT_TYPE_DISCONNECT:
            if(event.data>=DISC_NUM) event.data = DISC_NONE;
            if(event.peer==connpeer)
            {
                conoutf(CON_ERROR, "\f3could not connect to server");
                abortconnect();
            }
            else
            {
                if(!discmillis || event.data)
                {
                    const char *msg = disconnectreason(event.data);
                    if(msg) conoutf(CON_ERROR, "\f3server network error, disconnecting (%s) ...", msg);
                    else conoutf(CON_ERROR, "\f3server network error, disconnecting...");
                }
                disconnect();
            }
            return;

        default:
            break;
    }
}

