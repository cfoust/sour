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
    SOCKET_EVENT_DISCONNECT
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
};

extern SocketChannel socketCtl;

#endif ///__SOCKET_INCLUDED
