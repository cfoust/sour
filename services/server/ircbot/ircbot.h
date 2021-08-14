/**

    header for the IRC bot used in the QServ sauerbraten server mod

**/

#ifndef __IRCBOT_INCLUDED
#define __IRCBOT_INCLUDED

//#include <vector>
#include "game.h"

struct IrcMsg
{
    char nick[32];
    char user[32];
    char host[64];
    char chan[32];
    char message[512];
    int is_ready;
};

class ircBot
{
    public:
        void init(void);
        int getSock();
        int speak(const char *fmt, ...);
        bool checkping(char *buff);
        void sendpong();
        void periodicpong(char *buff);
        bool IsCommand(char *buff);
        void join(char *channel);
        void part(char *channel);
        void notice(char *user, const char *message);
        IrcMsg *lastmsg();
        hashtable<char *, int> IRCusers;
		
		bool isConnected();
    private:
        void ParseMessage(char *buff);
        int sock;
        IrcMsg msg;
        bool connected;
};

extern ircBot irc;

extern bool isloggedin(bool echo = 1);

#endif ///__IRCBOT_INCLUDED
