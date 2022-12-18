/**

    header for the IRC bot used in the QServ sauerbraten server mod

**/

#ifndef __SOCKET_INCLUDED
#define __SOCKET_INCLUDED

//#include <vector>
#include "game.h"

enum
{
    SOCKET_EVENT_CONNECT = 0,
    SOCKET_EVENT_RECEIVE,
    SOCKET_EVENT_DISCONNECT,
    SOCKET_EVENT_COMMAND,
	SOCKET_EVENT_RESPOND_MAP
};

enum
{
	SERVER_EVENT_PACKET = 0,
	SERVER_EVENT_BROADCAST,
	SERVER_EVENT_DISCONNECT,
	SERVER_EVENT_REQUEST_MAP,
	SERVER_EVENT_HEALTHY
};

class SocketChannel
{
    public:
        void init(void);
        int getSock();

		bool isConnected();
		void checkConnection();
		int send(char * data, int length);
		void finish();

        int receive(ENetPacket *packet);

    private:
        int sockFd, clientFd;
        bool connected;
        unsigned char buffer[5242880]; // 5MB

        int preconnectOffset = 0;
        // Used to buffer anything sent before we connected
        char preconnect[4096];
};

extern SocketChannel socketCtl;

#endif ///__SOCKET_INCLUDED
