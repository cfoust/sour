//#include "game.h"
#ifndef WIN32
#include <sys/socket.h>
#include <arpa/inet.h>
#include <netdb.h>
#include <sys/un.h>
#include <netinet/in.h>
#endif
#include "socket.h"
#include "../mod/QServ.h"

SVAR(socketpath, "/tmp/qserv_sock");

ICOMMAND(forceintermission, "", (), {
    server::startintermission();
});

ICOMMAND(kick, "i", (int *i), {
    disconnect_client(*i, DISC_KICK);
});

socketControl socketCtl;

int socketControl::getSock()
{
    return sock;
}

#include <unistd.h>
#include <stdio.h>

const int NUM_SECONDS = 10;

int ircstring = 0;
void socketControl::init()
{
    int con;
    char command[1000];

    struct sockaddr_un sa;
    struct hostent *he;

    sock = socket(AF_UNIX, SOCK_STREAM, 0);

    memset(&sa, 0, sizeof(struct sockaddr_un));
    sa.sun_family = AF_UNIX;
    strncpy(sa.sun_path, socketpath, sizeof(sa.sun_path) - 1);

    int result = bind(sock, (struct sockaddr *) &sa, sizeof(struct sockaddr_un));
    if (result == -1) {
        printf("Failed to bind to socket %s\n", socketpath);
        return;
    }

    printf("[ OK ] Initalizing socket control...\n");

    result = listen(sock, 5);
    if (result == -1) {
        printf("Failed to listen on socket %s\n", socketpath);
        return;
    }

    ssize_t numBytes;
    while(1) {
        int client = accept(sock, NULL, NULL);

        while ((numBytes = read(client, command, sizeof(command))) > 0) {
            printf("socket command: %s\n", command);
            execute(command);
        }

        memset(command, '\0', 1000);
    }
}
