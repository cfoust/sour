/*
 * A WebSocket to TCP socket proxy with support for "wss://" encryption.
 * Copyright 2010 Joel Martin
 * Licensed under LGPL version 3 (see docs/LICENSE.LGPL-3)
 *
 * You can make a cert/key with openssl using:
 * openssl req -new -x509 -days 365 -nodes -out self.pem -keyout self.pem
 * as taken from http://docs.python.org/dev/library/ssl.html#certificates
 */
#include <stdio.h>
#include <errno.h>
#include <limits.h>
#include <getopt.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netdb.h>
#include <sys/select.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <signal.h>
#include "websocket.h"

char traffic_legend[] = "\n\
Traffic Legend:\n\
    }  - Client receive\n\
    }. - Client receive partial\n\
    {  - Target receive\n\
\n\
    >  - Target send\n\
    >. - Target send partial\n\
    <  - Client send\n\
    <. - Client send partial\n\
";

char USAGE[] = "Usage: wsproxy [options] " \
               "[source_addr:]source_port\n\n" \
               "  --verbose|-v         verbose messages and per frame traffic\n" \
               "  --daemon|-d          become a daemon (background process)\n" \
               "  --whitelist-hosts|-W LIST  new-line separated target host whitelist file\n" \
               "  --whitelist-ports|-P LIST  new-line separated target port whitelist file\n" \
    
    
               "  --pid|-p             desired path of pid file. Default: '/var/run/websockify.pid'";

#define usage(fmt, args...) \
    do { \
        fprintf(stderr, "%s\n\n", USAGE); \
        fprintf(stderr, fmt , ## args); \
        exit(1); \
    } while(0)

char target_host[256] = "0.0.0.0";
int target_port = 28786;
int *target_ports;
int *target_hosts;
extern pipe_error;
extern settings_t settings;

void do_proxy(ws_ctx_t *ws_ctx, int target) {
    fd_set rlist, wlist, elist;
    struct timeval tv;
    int i, maxfd, client = ws_ctx->sockfd;
    unsigned int opcode, left, ret;
    unsigned int tout_start, tout_end, cout_start, cout_end;
    unsigned int tin_start, tin_end;
    ssize_t len, bytes;

    tout_start = tout_end = cout_start = cout_end;
    tin_start = tin_end = 0;
    maxfd = client > target ? client+1 : target+1;

    while (1) {
        tv.tv_sec = 1;
        tv.tv_usec = 0;

        FD_ZERO(&rlist);
        FD_ZERO(&wlist);
        FD_ZERO(&elist);

        FD_SET(client, &elist);
        FD_SET(target, &elist);

        if (tout_end == tout_start) {
            // Nothing queued for target, so read from client
            FD_SET(client, &rlist);
        } else {
            // Data queued for target, so write to it
            FD_SET(target, &wlist);
        }
        if (cout_end == cout_start) {
            // Nothing queued for client, so read from target
            FD_SET(target, &rlist);
        } else {
            // Data queued for client, so write to it
            FD_SET(client, &wlist);
        }

        ret = select(maxfd, &rlist, &wlist, &elist, &tv);
        if (pipe_error) { break; }

        if (FD_ISSET(target, &elist)) {
            handler_emsg("target exception\n");
            break;
        }
        if (FD_ISSET(client, &elist)) {
            handler_emsg("client exception\n");
            break;
        }

        if (ret == -1) {
            handler_emsg("select(): %s\n", strerror(errno));
            break;
        } else if (ret == 0) {
            //handler_emsg("select timeout\n");
            continue;
        }

        if (FD_ISSET(target, &wlist)) {
            len = tout_end-tout_start;
            bytes = sendto(target, ws_ctx->tout_buf + tout_start, len, 0, &ws_ctx->udpaddr, sizeof(ws_ctx->udpaddr));
            if (pipe_error) { break; }
            if (bytes < 0) {
                handler_emsg("target connection error: %s\n",
                             strerror(errno));
                break;
            }
            tout_start += bytes;
            if (tout_start >= tout_end) {
                tout_start = tout_end = 0;
                traffic(">");
            } else {
                traffic(">.");
            }
        }

        if (FD_ISSET(client, &wlist)) {
            len = cout_end-cout_start;
            bytes = ws_send(ws_ctx, ws_ctx->cout_buf + cout_start, len);
            if (pipe_error) { break; }
            if (len < 3) {
                handler_emsg("len: %d, bytes: %d: %d\n",
                             (int) len, (int) bytes,
                             (int) *(ws_ctx->cout_buf + cout_start));
            }
            cout_start += bytes;
            if (cout_start >= cout_end) {
                cout_start = cout_end = 0;
                traffic("<");
            } else {
                traffic("<.");
            }
        }

        if (FD_ISSET(target, &rlist)) {
            bytes = recv(target, ws_ctx->cin_buf, DBUFSIZE , 0);
            if (pipe_error) { break; }
            if (bytes <= 0) {
                handler_emsg("target closed connection\n");
                break;
            }
            cout_start = 0;
            if (ws_ctx->hybi) {
                cout_end = encode_hybi(ws_ctx->cin_buf, bytes,
                                   ws_ctx->cout_buf, BUFSIZE, ws_ctx->opcode);
            } else {
                cout_end = encode_hixie(ws_ctx->cin_buf, bytes,
                                    ws_ctx->cout_buf, BUFSIZE);
            }
            /*
            printf("encoded: ");
            for (i=0; i< cout_end; i++) {
                printf("%u,", (unsigned char) *(ws_ctx->cout_buf+i));
            }
            printf("\n");
            */
            if (cout_end < 0) {
                handler_emsg("encoding error\n");
                break;
            }
            traffic("{");
        }

        if (FD_ISSET(client, &rlist)) {
            bytes = ws_recv(ws_ctx, ws_ctx->tin_buf + tin_end, BUFSIZE-1);
            if (pipe_error) { break; }
            if (bytes <= 0) {
                handler_emsg("client closed connection\n");
                break;
            }
            tin_end += bytes;
            /*
            printf("before decode: ");
            for (i=0; i< bytes; i++) {
                printf("%u,", (unsigned char) *(ws_ctx->tin_buf+i));
            }
            printf("\n");
            */
            if (ws_ctx->hybi) {
                len = decode_hybi(ws_ctx->tin_buf + tin_start,
                                  tin_end-tin_start,
                                  ws_ctx->tout_buf, BUFSIZE-1,
                                  &opcode, &left);
            } else {
                len = decode_hixie(ws_ctx->tin_buf + tin_start,
                                   tin_end-tin_start,
                                   ws_ctx->tout_buf, BUFSIZE-1,
                                   &opcode, &left);
            }

            if (opcode == 8) {
                handler_msg("client sent orderly close frame\n");
                break;
            }

            /*
            printf("decoded: ");
            for (i=0; i< len; i++) {
                printf("%u,", (unsigned char) *(ws_ctx->tout_buf+i));
            }
            printf("\n");
            */

            if (len < 0) {
                handler_emsg("decoding error\n");
                break;
            }
            if (left) {
                tin_start = tin_end - left;
                //printf("partial frame from client");
            } else {
                tin_start = 0;
                tin_end = 0;
            }

            traffic("}");
            tout_start = 0;
            tout_end = len;
        }
    }
}

void proxy_handler(ws_ctx_t *ws_ctx) {
    int tsock = 0;
    struct sockaddr_in taddr = {0};
    struct sockaddr_in	addr = {0};
    char protocol = 'u';
    char dummy;

    sscanf(ws_ctx->headers->path+1, "%c%c%[^:]%c%d" , &protocol, &dummy, &target_host, &dummy,&target_port);

    if (target_ports != NULL) {
        int *p;
        int found = 0;
        for (p = target_ports; *p; p++) {
            if (*p == target_port) {
                found = 1;
                break;
            }
        }
        if (!found) {
            handler_emsg("Rejecting connection to non-whitelisted port: '%d'\n",
                         target_port);
            return;
        }
    }

    /* Resolve target address */
    if (resolve_host(&taddr.sin_addr, target_host) < -1) {
        handler_emsg("Could not resolve target address: %s\n",
                     strerror(errno));
    }

    if (target_hosts != NULL) {
        int *p;
        int found = 0;
        for (p = target_hosts; *p; p++) {
            if (*p == *((int*)&taddr.sin_addr)) {
                found = 1;
                break;
            }
        }
        if (!found) {
            handler_emsg("Rejecting connection to non-whitelisted host: '%s'\n",
                         target_host);
            return;
        }
    }

    handler_msg("connecting to: %s:%d via %c\n", target_host, target_port, protocol);

    if( protocol == 'u' )
        tsock = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
    else
        tsock = socket(AF_INET, SOCK_STREAM, 0);

    if (tsock < 0) {
        handler_emsg("Could not create target socket: %s\n",
                     strerror(errno));
        return;
    }
    //bzero((char *) &taddr, sizeof(taddr));
    taddr.sin_family = AF_INET;
    taddr.sin_port = htons(target_port);

    if ( protocol == 't' ) {
    if (connect(tsock, (struct sockaddr *) &taddr, sizeof(taddr)) < 0) {
        handler_emsg("Could not connect to target: %s\n",
                     strerror(errno));
        close(tsock);
        return;
    }
    }
    else {
    addr.sin_addr.s_addr = INADDR_ANY;
    addr.sin_port = 0;
    addr.sin_family = AF_INET;
    if( bind( tsock, (void *)&addr, sizeof( addr ) ) < 0 )
    {
        handler_emsg("Could not bind udp socket: %s\n",
                     strerror(errno));
        close(tsock);
        return;
    }
    ws_ctx->udpaddr = taddr;
    ws_ctx->udp = 1;
    }

    if ((settings.verbose) && (! settings.daemon)) {
        printf("%s", traffic_legend);
    }

    do_proxy(ws_ctx, tsock);

    shutdown(tsock, SHUT_RDWR);
    close(tsock);
}

int load_whitelist_port() {
  printf("loading port whitelist '%s'\n", settings.whitelist_port);
  FILE *whitelist = fopen(settings.whitelist_port, "r");
  if (whitelist == NULL) {
    fprintf(stderr, "Error opening whitelist file '%s':\n\t%s\n",
          settings.whitelist_port, strerror(errno));
    return -1;
  }

  const int tplen_grow = 512;
  int tplen = tplen_grow, tpcount = 0;
  target_ports = (int*)malloc(tplen*sizeof(int));
  if (target_ports == NULL) {
    fprintf(stderr, "Whitelist port malloc error");
    return -2;
  }

  char *line = NULL;
  ssize_t n = 0, nread = 0;
  while ((nread = getline(&line, &n, whitelist)) > 0) {
      if (line[0] == '\n') continue;
      line[nread-1] = '\x00';
      long int port = strtol(line, NULL, 10);
      if (port < 1 || port > 65535) {
          fprintf(stderr,
            "Whitelist port '%s' is not between valid range 1 and 65535", line);
          return -3;
      }
      tpcount++;
      if (tpcount >= tplen) {
          tplen += tplen_grow;
          target_ports = (int*)realloc(target_ports, tplen*sizeof(int));
          if (target_ports == NULL) {
              fprintf(stderr, "Whitelist port realloc error");
              return -2;
          }
      }
      target_ports[tpcount-1] = port;
  }
  if (line != NULL) free(line);

  if (tpcount == 0) {
      fprintf(stderr, "0 ports read from whitelist file '%s'\n",
                      settings.whitelist_port);
      return -4;
  }

  target_ports = (int*)realloc(target_ports, (tpcount + 1)*sizeof(int));
  if (target_ports == NULL) {
      fprintf(stderr, "Whitelist port realloc error");
      return -2;
  }
  target_ports[tpcount] = 0;
  return 0;
}

int load_whitelist_host() {
  printf("loading host whitelist '%s'\n", settings.whitelist_host);
  FILE *whitelist = fopen(settings.whitelist_host, "r");
  if (whitelist == NULL) {
    fprintf(stderr, "Error opening whitelist file '%s':\n\t%s\n",
          settings.whitelist_host, strerror(errno));
    return -1;
  }

  const int tplen_grow = 512;
  int tplen = tplen_grow, tpcount = 0;
  target_hosts = (int*)malloc(tplen*sizeof(int));
  if (target_hosts == NULL) {
    fprintf(stderr, "Whitelist port malloc error");
    return -2;
  }

  char *line = NULL;
  ssize_t n = 0, nread = 0;
  while ((nread = getline(&line, &n, whitelist)) > 0) {
      if (line[0] == '\n') continue;
      line[nread-1] = '\x00';
      int host;
      
      if (resolve_host(&host, line) < -1 ) {
          fprintf(stderr,
            "Whitelist host '%s': failed to resolve\n", line);
          //return -3;
          continue;
      }
      tpcount++;
      if (tpcount >= tplen) {
          tplen += tplen_grow;
          target_hosts = (int*)realloc(target_hosts, tplen*sizeof(int));
          if (target_hosts == NULL) {
              fprintf(stderr, "Whitelist port realloc error\n");
              return -2;
          }
      }
      target_hosts[tpcount-1] = host;
  }
  if (line != NULL) free(line);

  if (tpcount == 0) {
      fprintf(stderr, "0 ports read from whitelist file '%s'\n",
                      settings.whitelist_port);
      return -4;
  }

  target_hosts = (int*)realloc(target_hosts, (tpcount + 1)*sizeof(int));
  if (target_hosts == NULL) {
      fprintf(stderr, "Whitelist port realloc error\n");
      return -2;
  }
  target_hosts[tpcount] = 0;
  return 0;
}

int main(int argc, char *argv[])
{
    int fd, c, option_index = 0;
    char *found;
    static struct option long_options[] = {
        {"verbose",   no_argument,       0,                 'v'},
        {"daemon",    no_argument,       0,                 'd'},
        /* ---- */
        {"whitelist-ports", required_argument, 0,           'P'},
        {"whitelist-hosts", required_argument, 0,           'W'},
        {"pid",       required_argument, 0,                 'p'},
        {0, 0, 0, 0}
    };

    settings.pattern = "/%d";
    settings.pid = "/var/run/websockify.pid";

    while (1) {
        c = getopt_long (argc, argv, "vdW:p:P:",
                         long_options, &option_index);

        /* Detect the end */
        if (c == -1) break;

        switch (c) {
            case 0:
                break; // ignore
            case 1:
                break; // ignore
            case 'v':
                settings.verbose = 1;
                break;
            case 'd':
                settings.daemon = 1;
                break;
            case 'W':
                settings.whitelist_host = realpath(optarg, NULL);
                if (! settings.whitelist_host) {
                    usage("No whitelist file at %s\n", optarg);
                }
                break;
            case 'P':
                settings.whitelist_port = realpath(optarg, NULL);
                if (! settings.whitelist_port) {
                    usage("No whitelist file at %s\n", optarg);
                }
                break;
            case 'p':
                settings.pid = optarg;
                break;
            default:
                usage(" ");
        }
    }

    if ((argc-optind) != 1) {
        usage("Invalid number of arguments\n");
    }

    found = strstr(argv[optind], ":");
    if (found) {
        memcpy(settings.listen_host, argv[optind], found-argv[optind]);
        settings.listen_port = strtol(found+1, NULL, 10);
    } else {
        settings.listen_host[0] = '\0';
        settings.listen_port = strtol(argv[optind], NULL, 10);
    }
    optind++;
    if (settings.listen_port == 0) {
        usage("Could not parse listen_port\n");
    }

    if (!found && settings.whitelist_host != NULL) {
        if (load_whitelist_host()) {
          usage("Whitelist hosts error.");
        }

    }

    if (!found && settings.whitelist_port != NULL) {
        if (load_whitelist_port()) {
          usage("Whitelist ports error.");
        }

    }
    
    //printf("  verbose: %d\n",   settings.verbose);
    //printf("  daemon: %d\n",    settings.daemon);
    //printf("  run_once: %d\n",  settings.run_once);

    settings.handler = proxy_handler;
    start_server();

}
