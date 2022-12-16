#ifndef QSERV_H_INCLUDED
#define QSERV_H_INCLUDED

//detect uppercase letters for bad words (fix filter)

#include "game.h"
#include "fpsgame.h"
#include "../GeoIP/libGeoIP/GeoIP.h"
#include <time.h>
#include <pthread.h>
#include "../socket/socket.h"
#include <iostream>
#include <sstream>


const bool olanguagewarn = false;
const int maxolangwarnings = 5;
const char commandprefix = '#';

// restrict modes certain modes only for a tournament, not coop etc.
static char *qserv_modenames[] = {
"ffa", "coop", "teamplay", "insta", "instateam", "effic",
    "efficteam", "tac", "tacteam", "capture", "regencapture",
    "ctf", "instactf", "protect", "instaprotect", "hold", "instahold",
    "efficctf", "efficprotect", "effichold", "collect", "instacollect", "efficcollect"
};


static string owords[] = {
	"fuck", "shit", "cunt", "bitch", "whore", "twat",
    "faggot", "retard", "pussy", "ass",
    "vagina", "slut", "nigger", "queer", "gaylord", "gay",
    "homosexual", "semen", "creampie", "dick",
    "handjob", "blowjob", "cock", "tits", "penis"
};

static char bunny[] = "\f1('\\_/')\n(\f5=\f1'.'\f5=\f1)\n('')_('')";

struct SCommand {
    char name[50];
    char desc[500];
    int priv;
    int id;
    void (*func)(int, char**, int);
    bool hasargs;
    int args;
};

namespace server {
    struct msg {
        int count;
        double time;
    };

    class QServ {
        public:
            QServ(bool, int, char);
            ~QServ();
            
            int instacoop_gamelimit;

            bool initgeoip(const char*);
            bool initcitygeoip(const char*);
            char *congeoip(const char*);
            //char *citygeoip(const char*);
            //char *regiongeoip(const char*);
        
            std::string cgip(const char*);

            void newcommand(const char*, const char*, int, void (*)(int, char**, int), int);
            bool isCommand(char*);
            int getCommand(char*, char**);
            void exeCommand(int, char**, int);

            char findWord(char*, char*, bool);

            char *getCommandName(int);
            char *getCommandDesc(int);
            int getCommandPriv(int);
            bool commandHasArgs(int);
            int getCommandArgCount(int);
            int getlastCommand();

            void checkoLang(int, char*);
            void setoLangWarn(int);
            void resetoLangWarn(int);
            int getoLangWarn(int);

            void initCommands(void (*)());

            void setFullText(const char*);
            char *getFullText();

            void setSender(int);
            int getSender();

            void setlastCI(clientinfo*);
            clientinfo getlastCI();

            void setlastSA(bool);
            bool getlastSA();
            char *cntoip(int);

            clientinfo *getClient(int);

            static clientinfo m_lastCI;

            bool handleTextCommands(clientinfo*, char*);

            bool isLangWarnOn();
            void setoLang(bool);

            void setCmdPrefix(unsigned char);
            char getCmdPrefix();

            void getLocation(clientinfo*);

            void checkMsg(int);
            int getMsgC(int);
            void resetMsg(int);
        protected:
            GeoIP *m_geoip;
            GeoIP *city_geoip;
            int m_lastcommand;
            SCommand m_command[50];
            int m_oLangWarn[1000];
            int m_oTimes;
            void (*m_checkolangCallback)(int, char*);
            char m_fulltext[1024];
            int m_sender;
            bool m_lastSA;
            int m_owordcount;
            bool m_olangcheck;
            int m_maxolangwarns;
            char m_cmdprefix;

            msg ms[1000];
    };
}

enum
{
    ECHO_ALL = 0,
    ECHO_IRC ,
    ECHO_CONSOLE,
    ECHO_SERV,
    ECHO_NOCOLOR,
};
extern void out(int type, const char *fmt, ...);

#define toip(cn) qs.cntoip(cn)

extern server::QServ qs;

extern int count;
extern int msgcount[128];

#endif
