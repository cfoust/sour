// implementation of generic tools

#include "cube.h"

#ifndef WIN32
#include <unistd.h>
#endif

int guessnumcpus()
{
    int numcpus = 1;
#ifdef WIN32
    SYSTEM_INFO info;
    GetSystemInfo(&info);
    numcpus = (int)info.dwNumberOfProcessors;
#elif defined(_SC_NPROCESSORS_ONLN)
    numcpus = (int)sysconf(_SC_NPROCESSORS_ONLN);
#endif
    return max(numcpus, 1);
}
    
////////////////////////// rnd numbers ////////////////////////////////////////

#define N (624)             
#define M (397)                
#define K (0x9908B0DFU)       

static uint state[N];
static int next = N;

void seedMT(uint seed)
{
    state[0] = seed;
    for(uint i = 1; i < N; i++)
        state[i] = seed = 1812433253U * (seed ^ (seed >> 30)) + i;
    next = 0;
}

void filtertext(char *dst, const char *src, bool whitespace, int len)
{
    for(int c = uchar(*src); c; c = uchar(*++src))
    {
        if(c == '\f')
        {
            if(!*++src) break;
            continue;
        }
        if(iscubeprint(c) || (iscubespace(c) && whitespace))
        {
            *dst++ = c;
            if(!--len) break;
        }
    }
    *dst = '\0';
}

static string tmpstr[4];
static int tmpidx = 0;
char *tempformatstring(const char *fmt, ...)
{
    tmpidx = (tmpidx+1)%4;
    
    va_list v;
    va_start(v, fmt);
    vformatstring(tmpstr[tmpidx], fmt, v);
    va_end(v);
    
    return tmpstr[tmpidx];
}

void ipmask::parse(const char *name)
{
    union { uchar b[sizeof(enet_uint32)]; enet_uint32 i; } ipconv, maskconv;
    ipconv.i = 0;
    maskconv.i = 0;
    loopi(4)
    {
        char *end = NULL;
        int n = strtol(name, &end, 10);
        if(!end) break;
        if(end > name) { ipconv.b[i] = n; maskconv.b[i] = 0xFF; }
        name = end;
        while(int c = *name)
        {
            ++name;
            if(c == '.') break;
            if(c == '/')
            {
                int range = clamp(int(strtol(name, NULL, 10)), 0, 32);
                mask = range ? ENET_HOST_TO_NET_32(0xFFffFFff << (32 - range)) : maskconv.i;
                ip = ipconv.i & mask;
                return;
            }
        }
    }
    ip = ipconv.i;
    mask = maskconv.i;
}

int ipmask::print(char *buf) const
{
    char *start = buf;
    union { uchar b[sizeof(enet_uint32)]; enet_uint32 i; } ipconv, maskconv;
    ipconv.i = ip;
    maskconv.i = mask;
    int lastdigit = -1;
    loopi(4) if(maskconv.b[i])
    {
        if(lastdigit >= 0) *buf++ = '.';
        loopj(i - lastdigit - 1) { *buf++ = '*'; *buf++ = '.'; }
        buf += sprintf(buf, "%d", ipconv.b[i]);
        lastdigit = i;
    }
    enet_uint32 bits = ~ENET_NET_TO_HOST_32(mask);
    int range = 32;
    for(; (bits&0xFF) == 0xFF; bits >>= 8) range -= 8;
    for(; bits&1; bits >>= 1) --range;
    if(!bits && range%8) buf += sprintf(buf, "/%d", range);
    return int(buf-start);
}


uint randomMT()
{
    int cur = next;
    if(++next >= N)
    {
        if(next > N) { seedMT(5489U + time(NULL)); cur = next++; }
        else next = 0;
    }
    uint y = (state[cur] & 0x80000000U) | (state[next] & 0x7FFFFFFFU);
    state[cur] = y = state[cur < N-M ? cur + M : cur + M-N] ^ (y >> 1) ^ (-int(y & 1U) & K);
    y ^= (y >> 11);
    y ^= (y <<  7) & 0x9D2C5680U;
    y ^= (y << 15) & 0xEFC60000U;
    y ^= (y >> 18);
    return y;
}
