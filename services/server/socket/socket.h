/**

    header for the IRC bot used in the QServ sauerbraten server mod

**/

#ifndef __SOCKET_INCLUDED
#define __SOCKET_INCLUDED

//#include <vector>
#include "game.h"

class socketControl
{
    public:
        void init(void);
        int getSock();
		
		bool isConnected();
    private:
        int sock;
        bool connected;
};

extern socketControl socketCtl;

extern bool isloggedin(bool echo = 1);

#endif ///__SOCKET_INCLUDED
