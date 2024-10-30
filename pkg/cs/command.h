#ifndef _TOOLS_H
#define _TOOLS_H
#define __cdecl
#define _vsnprintf vsnprintf

#include <assert.h>
#include <ctype.h>
#include <limits.h>
#include <math.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#ifdef NULL
#undef NULL
#endif
#define NULL 0

#define PATHDIV '/'

typedef unsigned char uchar;
typedef unsigned short ushort;
typedef unsigned int uint;
typedef signed long long int llong;
typedef unsigned long long int ullong;

#ifdef _DEBUG
#define ASSERT(c) assert(c)
#else
#define ASSERT(c) if(c) {}
#endif

void *operator new(size_t, bool);
void *operator new[](size_t, bool);
inline void *operator new(size_t, void *p) { return p; }
inline void *operator new[](size_t, void *p) { return p; }
inline void operator delete(void *, void *) {}
inline void operator delete[](void *, void *) {}

static inline uint hthash(const char *key)
{
    uint h = 5381;
    for(int i = 0, k; (k = key[i]); i++) h = ((h<<5)+h)^k;    // bernstein k=33 xor
    return h;
}

static inline bool htcmp(const char *x, const char *y)
{
    return !strcmp(x, y);
}

#ifdef __GNUC__
#define PRINTFARGS(fmt, args) __attribute__((format(printf, fmt, args)))
#else
#define PRINTFARGS(fmt, args)
#endif

#define DELETEP(p) if(p) { delete   p; p = 0; }
#define DELETEA(p) if(p) { delete[] p; p = 0; }

#define rnd(x) ((int)(randomMT()&0x7FFFFFFF)%(x))
#define rndscale(x) (float((randomMT()&0x7FFFFFFF)*double(x)/double(0x7FFFFFFF)))
#define detrnd(s, x) ((int)(((((uint)(s))*1103515245+12345)>>16)%(x)))

#define loop(v,m) for(int v = 0; v < int(m); ++v)
#define loopi(m) loop(i,m)
#define loopj(m) loop(j,m)
#define loopk(m) loop(k,m)
#define loopl(m) loop(l,m)
#define looprev(v,m) for(int v = int(m); --v >= 0;)
#define loopirev(m) looprev(i,m)
#define loopjrev(m) looprev(j,m)
#define loopkrev(m) looprev(k,m)
#define looplrev(m) looprev(l,m)
#define loopv(v)    for(int i = 0; i<(v).length(); i++)
#define loopvj(v)   for(int j = 0; j<(v).length(); j++)
#define loopvk(v)   for(int k = 0; k<(v).length(); k++)
#define loopvrev(v) for(int i = (v).length()-1; i>=0; i--)

#define defformatstring(d,...) string d; formatstring(d, __VA_ARGS__)
#define defvformatstring(d,last,fmt) string d; { va_list ap; va_start(ap, last); vformatstring(d, fmt, ap); va_end(ap); }

inline void vformatstring(char *d, const char *fmt, va_list v, int len) { _vsnprintf(d, len, fmt, v); d[len-1] = 0; }
template<size_t N> inline void vformatstring(char (&d)[N], const char *fmt, va_list v) { vformatstring(d, fmt, v, N); }

template<size_t N> inline void formatstring(char (&d)[N], const char *fmt, ...) PRINTFARGS(2, 3);
template<size_t N> inline void formatstring(char (&d)[N], const char *fmt, ...)
{
    va_list v;
    va_start(v, fmt);
    vformatstring(d, fmt, v, int(N));
    va_end(v);
}

inline void nformatstring(char *d, int len, const char *fmt, ...) PRINTFARGS(3, 4);
inline void nformatstring(char *d, int len, const char *fmt, ...)
{
    va_list v;
    va_start(v, fmt);
    vformatstring(d, fmt, v, len);
    va_end(v);
}

static inline void intformat(char *buf, int v, int len = 20) { nformatstring(buf, len, "%d", v); }
static inline void floatformat(char *buf, float v, int len = 20) { nformatstring(buf, len, v==int(v) ? "%.1f" : "%.6g", v); }

#define MAXSTRLEN 260
typedef char string[MAXSTRLEN];

#ifdef __GNUC__
#define UNUSED __attribute__((unused))
#else
#define UNUSED
#endif

template<class T, class F>
static inline void quicksort(T *start, T *end, F fun)
{
    while(end-start > 10)
    {
        T *mid = &start[(end-start)/2], *i = start+1, *j = end-2, pivot;
        if(fun(*start, *mid)) /* start < mid */
        {
            if(fun(end[-1], *start)) { pivot = *start; *start = end[-1]; end[-1] = *mid; } /* end < start < mid */
            else if(fun(end[-1], *mid)) { pivot = end[-1]; end[-1] = *mid; } /* start <= end < mid */
            else { pivot = *mid; } /* start < mid <= end */
        }
        else if(fun(*start, end[-1])) { pivot = *start; *start = *mid; } /*mid <= start < end */
        else if(fun(*mid, end[-1])) { pivot = end[-1]; end[-1] = *start; *start = *mid; } /* mid < end <= start */
        else { pivot = *mid; swap(*start, end[-1]); }  /* end <= mid <= start */
        *mid = end[-2];
        do
        {
            while(fun(*i, pivot)) if(++i >= j) goto partitioned;
            while(fun(pivot, *--j)) if(i >= j) goto partitioned;
            swap(*i, *j);
        }
        while(++i < j);
    partitioned:
        end[-2] = *i;
        *i = pivot;
        
        if(i-start < end-(i+1))
        {
            quicksort(start, i, fun);
            start = i+1;
        }
        else
        {
            quicksort(i+1, end, fun);
            end = i;
        }
    }
    
    insertionsort(start, end, fun);
}

template<class T, class F>
static inline void quicksort(T *buf, int n, F fun)
{
    quicksort(buf, buf+n, fun);
}

template<class T>
static inline bool compareless(const T &x, const T &y) { return x < y; }

template<class T, class F>
static inline void insertionsort(T *start, T *end, F fun)
{
    for(T *i = start+1; i < end; i++)
    {
        if(fun(*i, i[-1]))
        {
            T tmp = *i;
            *i = i[-1];
            T *j = i-1;
            for(; j > start && fun(tmp, j[-1]); --j)
                *j = j[-1];
            *j = tmp;
        }
    }
    
}

template<class T>
static inline void quicksort(T *buf, int n)
{
    quicksort(buf, buf+n, compareless<T>);
}


struct sortless
{
    template<class T> bool operator()(const T &x, const T &y) const { return x < y; }
    bool operator()(char *x, char *y) const { return strcmp(x, y) < 0; }
    bool operator()(const char *x, const char *y) const { return strcmp(x, y) < 0; }
};

struct sortnameless
{
    template<class T> bool operator()(const T &x, const T &y) const { return sortless()(x.name, y.name); }
    template<class T> bool operator()(T *x, T *y) const { return sortless()(x->name, y->name); }
    template<class T> bool operator()(const T *x, const T *y) const { return sortless()(x->name, y->name); }
};

template<class T>
static inline T max(T a, T b)
{
    return a > b ? a : b;
}
template<class T>
static inline T min(T a, T b)
{
    return a < b ? a : b;
}
template<class T, class U>
static inline T clamp(T a, U b, U c)
{
    return max(T(b), min(a, T(c)));
}

template<class T>
static inline void swap(T &a, T &b)
{
    T t = a;
    a = b;
    b = t;
}

typedef signed char schar;
typedef unsigned char uchar;
typedef unsigned short ushort;
typedef unsigned int uint;
typedef unsigned long ulong;
typedef signed long long int llong;
typedef unsigned long long int ullong;

template <class T>
struct databuf
{
    enum
    {
        OVERREAD  = 1<<0,
        OVERWROTE = 1<<1
    };

    T *buf;
    int len, maxlen;
    uchar flags;

    databuf() : buf(NULL), len(0), maxlen(0), flags(0) {}

    template<class U>
    databuf(T *buf, U maxlen) : buf(buf), len(0), maxlen((int)maxlen), flags(0) {}

    void reset()
    {
        len = 0;
        flags = 0;
    }

    void reset(T *buf_, int maxlen_)
    {
        reset();
        buf = buf_;
        maxlen = maxlen_;
    }

    const T &get()
    {
        static T overreadval = 0;
        if(len<maxlen) return buf[len++];
        flags |= OVERREAD;
        return overreadval;
    }

    databuf subbuf(int sz)
    {
        sz = clamp(sz, 0, maxlen-len);
        len += sz;
        return databuf(&buf[len-sz], sz);
    }

    T *pad(int numvals)
    {
        T *vals = &buf[len];
        len += min(numvals, maxlen-len);
        return vals;
    }

    void put(const T &val)
    {
        if(len<maxlen) buf[len++] = val;
        else flags |= OVERWROTE;
    }

    void put(const T *vals, int numvals)
    {
        if(maxlen-len<numvals) flags |= OVERWROTE;
        memcpy(&buf[len], (const void *)vals, min(maxlen-len, numvals)*sizeof(T));
        len += min(maxlen-len, numvals);
    }

    int get(T *vals, int numvals)
    {
        int read = min(maxlen-len, numvals);
        if(read<numvals) flags |= OVERREAD;
        memcpy(vals, (void *)&buf[len], read*sizeof(T));
        len += read;
        return read;
    }

    void offset(int n)
    {
        n = min(n, maxlen);
        buf += n;
        maxlen -= n;
        len = max(len-n, 0);
    }

    T *getbuf() const { return buf; }
    bool empty() const { return len==0; }
    int length() const { return len; }
    int remaining() const { return maxlen-len; }
    bool overread() const { return (flags&OVERREAD)!=0; }
    bool overwrote() const { return (flags&OVERWROTE)!=0; }

    bool check(int n) { return remaining() >= n; }

    void forceoverread()
    {
        len = maxlen;
        flags |= OVERREAD;
    }
};

typedef databuf<char> charbuf;
typedef databuf<uchar> ucharbuf;


template <class T> struct vector
{
    static const int MINSIZE = 8;

    T *buf;
    int alen, ulen;

    vector() : buf(NULL), alen(0), ulen(0)
    {
    }

    vector(const vector &v) : buf(NULL), alen(0), ulen(0)
    {
        *this = v;
    }

    ~vector() { shrink(0); if(buf) delete[] (uchar *)buf; }

    vector<T> &operator=(const vector<T> &v)
    {
        shrink(0);
        if(v.length() > alen) growbuf(v.length());
        loopv(v) add(v[i]);
        return *this;
    }

    T &add(const T &x)
    {
        if(ulen==alen) growbuf(ulen+1);
        new (&buf[ulen]) T(x);
        return buf[ulen++];
    }

    T &add()
    {
        if(ulen==alen) growbuf(ulen+1);
        new (&buf[ulen]) T;
        return buf[ulen++];
    }

    T &dup()
    {
        if(ulen==alen) growbuf(ulen+1);
        new (&buf[ulen]) T(buf[ulen-1]);
        return buf[ulen++];
    }

    void move(vector<T> &v)
    {
        if(!ulen)
        {
            swap(buf, v.buf);
            swap(ulen, v.ulen);
            swap(alen, v.alen);
        }
        else
        {
            growbuf(ulen+v.ulen);
            if(v.ulen) memcpy(&buf[ulen], (void  *)v.buf, v.ulen*sizeof(T));
            ulen += v.ulen;
            v.ulen = 0;
        }
    }

    bool inrange(size_t i) const { return i<size_t(ulen); }
    bool inrange(int i) const { return i>=0 && i<ulen; }

    T &pop() { return buf[--ulen]; }
    T &last() { return buf[ulen-1]; }
    void drop() { ulen--; buf[ulen].~T(); }
    bool empty() const { return ulen==0; }

    int capacity() const { return alen; }
    int length() const { return ulen; }
    T &operator[](int i) { ASSERT(i>=0 && i<ulen); return buf[i]; }
    const T &operator[](int i) const { ASSERT(i >= 0 && i<ulen); return buf[i]; }

    void disown() { buf = NULL; alen = ulen = 0; }

    // XXX always isclass=false
    void shrink(int i) { ASSERT(i<=ulen); if(false) ulen = i; else while(ulen>i) drop(); }
    void setsize(int i) { ASSERT(i<=ulen); ulen = i; }

    void deletecontents() { while(!empty()) delete   pop(); }
    void deletearrays() { while(!empty()) delete[] pop(); }

    T *getbuf() { return buf; }
    const T *getbuf() const { return buf; }
    bool inbuf(const T *e) const { return e >= buf && e < &buf[ulen]; }

    template<class F>
    void sort(F fun, int i = 0, int n = -1)
    {
        quicksort(&buf[i], n < 0 ? ulen-i : n, fun);
    }

    void sort() { sort(sortless()); }
    void sortname() { sort(sortnameless()); }

    void growbuf(int sz)
    {
        int olen = alen;
        if(alen <= 0) alen = max(MINSIZE, sz);
        else while(alen < sz) alen += alen/2;
        if(alen <= olen) return;
        uchar *newbuf = new uchar[alen*sizeof(T)];
        if(olen > 0)
        {
            if(ulen > 0) memcpy(newbuf, (void *)buf, ulen*sizeof(T));
            delete[] (uchar *)buf;
        }
        buf = (T *)newbuf;
    }

    databuf<T> reserve(int sz)
    {
        if(alen-ulen < sz) growbuf(ulen+sz);
        return databuf<T>(&buf[ulen], sz);
    }

    void advance(int sz)
    {
        ulen += sz;
    }

    void addbuf(const databuf<T> &p)
    {
        advance(p.length());
    }

    T *pad(int n)
    {
        T *buf = reserve(n).buf;
        advance(n);
        return buf;
    }

    void put(const T &v) { add(v); }

    void put(const T *v, int n)
    {
        databuf<T> buf = reserve(n);
        buf.put(v, n);
        addbuf(buf);
    }

    void remove(int i, int n)
    {
        for(int p = i+n; p<ulen; p++) buf[p-n] = buf[p];
        ulen -= n;
    }

    T remove(int i)
    {
        T e = buf[i];
        for(int p = i+1; p<ulen; p++) buf[p-1] = buf[p];
        ulen--;
        return e;
    }

    T removeunordered(int i)
    {
        T e = buf[i];
        ulen--;
        if(ulen>0) buf[i] = buf[ulen];
        return e;
    }

    template<class U>
    int find(const U &o)
    {
        loopi(ulen) if(buf[i]==o) return i;
        return -1;
    }

    void addunique(const T &o)
    {
        if(find(o) < 0) add(o);
    }

    void removeobj(const T &o)
    {
        loopi(ulen) if(buf[i] == o)
        {
            int dst = i;
            for(int j = i+1; j < ulen; j++) if(!(buf[j] == o)) buf[dst++] = buf[j];
            setsize(dst);
            break;
        }
    }

    void replacewithlast(const T &o)
    {
        if(!ulen) return;
        loopi(ulen-1) if(buf[i]==o)
        {
            buf[i] = buf[ulen-1];
            break;
        }
        ulen--;
    }

    T &insert(int i, const T &e)
    {
        add(T());
        for(int p = ulen-1; p>i; p--) buf[p] = buf[p-1];
        buf[i] = e;
        return buf[i];
    }

    T *insert(int i, const T *e, int n)
    {
        if(alen-ulen < n) growbuf(ulen+n);
        loopj(n) add(T());
        for(int p = ulen-1; p>=i+n; p--) buf[p] = buf[p-n];
        loopj(n) buf[i+j] = e[j];
        return &buf[i];
    }

    void reverse()
    {
        loopi(ulen/2) swap(buf[i], buf[ulen-1-i]);
    }

    static int heapparent(int i) { return (i - 1) >> 1; }
    static int heapchild(int i) { return (i << 1) + 1; }

    void buildheap()
    {
        for(int i = ulen/2; i >= 0; i--) downheap(i);
    }

    int upheap(int i)
    {
        float score = heapscore(buf[i]);
        while(i > 0)
        {
            int pi = heapparent(i);
            if(score >= heapscore(buf[pi])) break;
            swap(buf[i], buf[pi]);
            i = pi;
        }
        return i;
    }

    T &addheap(const T &x)
    {
        add(x);
        return buf[upheap(ulen-1)];
    }

    int downheap(int i)
    {
        float score = heapscore(buf[i]);
        for(;;)
        {
            int ci = heapchild(i);
            if(ci >= ulen) break;
            float cscore = heapscore(buf[ci]);
            if(score > cscore)
            {
               if(ci+1 < ulen && heapscore(buf[ci+1]) < cscore) { swap(buf[ci+1], buf[i]); i = ci+1; }
               else { swap(buf[ci], buf[i]); i = ci; }
            }
            else if(ci+1 < ulen && heapscore(buf[ci+1]) < score) { swap(buf[ci+1], buf[i]); i = ci+1; }
            else break;
        }
        return i;
    }

    T removeheap()
    {
        T e = removeunordered(0);
        if(ulen) downheap(0);
        return e;
    }

    template<class K> 
    int htfind(const K &key)
    {
        loopi(ulen) if(htcmp(key, buf[i])) return i;
        return -1;
    }
};

template<class H, class E, class K, class T> struct hashbase
{
    typedef E elemtype;
    typedef K keytype;
    typedef T datatype;

    enum { CHUNKSIZE = 64 };

    struct chain { E elem; chain *next; };
    struct chainchunk { chain chains[CHUNKSIZE]; chainchunk *next; };

    int size;
    int numelems;
    chain **chains;

    chainchunk *chunks;
    chain *unused;

    enum { DEFAULTSIZE = 1<<10 };

    hashbase(int size = DEFAULTSIZE)
      : size(size)
    {
        numelems = 0;
        chunks = NULL;
        unused = NULL;
        chains = new chain *[size];
        memset(chains, 0, size*sizeof(chain *));
    }

    ~hashbase()
    {
        DELETEA(chains);
        deletechunks();
    }

    chain *insert(uint h)
    {
        if(!unused)
        {
            chainchunk *chunk = new chainchunk;
            chunk->next = chunks;
            chunks = chunk;
            loopi(CHUNKSIZE-1) chunk->chains[i].next = &chunk->chains[i+1];
            chunk->chains[CHUNKSIZE-1].next = unused;
            unused = chunk->chains;
        }
        chain *c = unused;
        unused = unused->next;
        c->next = chains[h];
        chains[h] = c;
        numelems++;
        return c;
    }

    template<class U>
    T &insert(uint h, const U &key)
    {
        chain *c = insert(h);
        H::setkey(c->elem, key);
        return H::getdata(c->elem);
    }

    #define HTFIND(success, fail) \
        uint h = hthash(key)&(this->size-1); \
        for(chain *c = this->chains[h]; c; c = c->next) \
        { \
            if(htcmp(key, H::getkey(c->elem))) return success H::getdata(c->elem); \
        } \
        return (fail);

    template<class U>
    T *access(const U &key)
    {
        HTFIND(&, NULL);
    }

    template<class U, class V>
    T &access(const U &key, const V &elem)
    {
        HTFIND( , insert(h, key) = elem);
    }

    template<class U>
    T &operator[](const U &key)
    {
        HTFIND( , insert(h, key));
    }

    template<class U>
    T &find(const U &key, T &notfound)
    {
        HTFIND( , notfound);
    }

    template<class U>
    const T &find(const U &key, const T &notfound)
    {
        HTFIND( , notfound);
    }

    template<class U>
    bool remove(const U &key)
    {
        uint h = hthash(key)&(size-1);
        for(chain **p = &chains[h], *c = chains[h]; c; p = &c->next, c = c->next)
        {
            if(htcmp(key, H::getkey(c->elem)))
            {
                *p = c->next;
                c->elem.~E();
                new (&c->elem) E;
                c->next = unused;
                unused = c;
                numelems--;
                return true;
            }
        }
        return false;
    }

    void deletechunks()
    {
        for(chainchunk *nextchunk; chunks; chunks = nextchunk)
        {
            nextchunk = chunks->next;
            delete chunks;
        }
    }

    void clear()
    {
        if(!numelems) return;
        memset(chains, 0, size*sizeof(chain *));
        numelems = 0;
        unused = NULL;
        deletechunks();
    }

    static inline chain *enumnext(void *i) { return ((chain *)i)->next; }
    static inline K &enumkey(void *i) { return H::getkey(((chain *)i)->elem); }
    static inline T &enumdata(void *i) { return H::getdata(((chain *)i)->elem); }
};

template<class T> struct hashset : hashbase<hashset<T>, T, T, T>
{
    typedef hashbase<hashset<T>, T, T, T> basetype;

    hashset(int size = basetype::DEFAULTSIZE) : basetype(size) {}

    static inline const T &getkey(const T &elem) { return elem; }
    static inline T &getdata(T &elem) { return elem; }
    template<class K> static inline void setkey(T &elem, const K &key) {}

    template<class V>
    T &add(const V &elem)
    {
        return basetype::access(elem, elem);
    }
};

template<class T> struct hashnameset : hashbase<hashnameset<T>, T, const char *, T>
{
    typedef hashbase<hashnameset<T>, T, const char *, T> basetype;

    hashnameset(int size = basetype::DEFAULTSIZE) : basetype(size) {}

    template<class U> static inline const char *getkey(const U &elem) { return elem.name; }
    template<class U> static inline const char *getkey(U *elem) { return elem->name; }
    static inline T &getdata(T &elem) { return elem; }
    template<class K> static inline void setkey(T &elem, const K &key) {}

    template<class V>
    T &add(const V &elem)
    {
        return basetype::access(getkey(elem), elem);
    }
};

template<class K, class T> struct hashtableentry
{
    K key;
    T data;
};

template<class K, class T> struct hashtable : hashbase<hashtable<K, T>, hashtableentry<K, T>, K, T>
{
    typedef hashbase<hashtable<K, T>, hashtableentry<K, T>, K, T> basetype;
    typedef typename basetype::elemtype elemtype;

    hashtable(int size = basetype::DEFAULTSIZE) : basetype(size) {}

    static inline K &getkey(elemtype &elem) { return elem.key; }
    static inline T &getdata(elemtype &elem) { return elem.data; }
    template<class U> static inline void setkey(elemtype &elem, const U &key) { elem.key = key; }
};

#define enumeratekt(ht,k,e,t,f,b) loopi((ht).size) for(void *ec = (ht).chains[i]; ec;) { k &e = (ht).enumkey(ec); t &f = (ht).enumdata(ec); ec = (ht).enumnext(ec); b; }
#define enumerate(ht,t,e,b)       loopi((ht).size) for(void *ec = (ht).chains[i]; ec;) { t &e = (ht).enumdata(ec); ec = (ht).enumnext(ec); b; }

inline char *copystring(char *d, const char *s, size_t len)
{
    size_t slen = min(strlen(s), len-1);
    memcpy(d, s, slen);
    d[slen] = 0;
    return d;
}
template<size_t N> inline char *copystring(char (&d)[N], const char *s) { return copystring(d, s, N); }

inline char *newstring(size_t l)                { return new char[l+1]; }
inline char *newstring(const char *s, size_t l) { return copystring(newstring(l), s, l+1); }
inline char *newstring(const char *s)           { size_t l = strlen(s); char *d = newstring(l); memcpy(d, s, l+1); return d; }

#define PI  (3.1415927f)
#define PI2 (2*PI)
#define SQRT2 (1.4142136f)
#define SQRT3 (1.7320508f)
#define RAD (PI / 180.0f)

// script binding functionality

enum { VAL_NULL = 0, VAL_INT, VAL_FLOAT, VAL_STR, VAL_ANY, VAL_CODE, VAL_MACRO, VAL_IDENT };

enum
{
    CODE_START = 0,
    CODE_OFFSET,
    CODE_POP,
    CODE_ENTER,
    CODE_EXIT,
    CODE_VAL,
    CODE_VALI,
    CODE_MACRO,
    CODE_BOOL,
    CODE_BLOCK,
    CODE_COMPILE,
    CODE_FORCE,
    CODE_RESULT,
    CODE_IDENT, CODE_IDENTU, CODE_IDENTARG,
    CODE_COM, CODE_COMD, CODE_COMC, CODE_COMV,
    CODE_CONC, CODE_CONCW, CODE_CONCM, CODE_DOWN,
    CODE_SVAR, CODE_SVAR1,
    CODE_IVAR, CODE_IVAR1, CODE_IVAR2, CODE_IVAR3,
    CODE_FVAR, CODE_FVAR1,
    CODE_LOOKUP, CODE_LOOKUPU, CODE_LOOKUPARG, CODE_ALIAS, CODE_ALIASU, CODE_ALIASARG, CODE_CALL, CODE_CALLU, CODE_CALLARG,
    CODE_PRINT,
    CODE_LOCAL,

    CODE_OP_MASK = 0x3F,
    CODE_RET = 6,
    CODE_RET_MASK = 0xC0,

    /* return type flags */
    RET_NULL   = VAL_NULL<<CODE_RET,
    RET_STR    = VAL_STR<<CODE_RET,
    RET_INT    = VAL_INT<<CODE_RET,
    RET_FLOAT  = VAL_FLOAT<<CODE_RET,
};

enum { ID_VAR, ID_FVAR, ID_SVAR, ID_COMMAND, ID_ALIAS, ID_LOCAL };

enum { IDF_PERSIST = 1<<0, IDF_OVERRIDE = 1<<1, IDF_HEX = 1<<2, IDF_READONLY = 1<<3, IDF_OVERRIDDEN = 1<<4, IDF_UNKNOWN = 1<<5, IDF_ARG = 1<<6, IDF_EMUVAR = 1<<7 };

struct ident;

struct identval
{
    union
    {
        int i;      // ID_VAR, VAL_INT
        float f;    // ID_FVAR, VAL_FLOAT
        char *s;    // ID_SVAR, VAL_STR
        const uint *code; // VAL_CODE
        ident *id;  // VAL_IDENT
    };
};

struct tagval : identval
{
    int type;

    void setint(int val) { type = VAL_INT; i = val; }
    void setfloat(float val) { type = VAL_FLOAT; f = val; }
    void setstr(char *val) { type = VAL_STR; s = val; }
    void setnull() { type = VAL_NULL; i = 0; }
    void setcode(const uint *val) { type = VAL_CODE; code = val; }
    void setmacro(const uint *val) { type = VAL_MACRO; code = val; }
    void setident(ident *val) { type = VAL_IDENT; id = val; }

    const char *getstr() const;
    int getint() const;
    float getfloat() const;
    bool getbool() const;

    void cleanup();
};

struct identstack
{
    identval val;
    int valtype;
    identstack *next;
};

union identvalptr
{
    int *i;   // ID_VAR
    float *f; // ID_FVAR
    char **s; // ID_SVAR
};

typedef void (__cdecl *identfun)();

struct ident
{
    uchar type; // one of ID_* above
    union
    {
        uchar valtype; // ID_ALIAS
        uchar numargs; // ID_COMMAND
    };
    ushort flags;
    int index;
    const char *name;
    union
    {
        struct // ID_VAR, ID_FVAR, ID_SVAR
        {
            union
            {
                struct { int minval, maxval; };     // ID_VAR
                struct { float minvalf, maxvalf; }; // ID_FVAR
            };
            identvalptr storage;
            identval overrideval;
        };
        struct // ID_ALIAS
        {
            uint *code;
            identval val;
            identstack *stack;
        };
        struct // ID_COMMAND
        {
            const char *args;
            uint argmask;
        };
    };
    identfun fun; // ID_VAR, ID_FVAR, ID_SVAR, ID_COMMAND
    
    ident() {}
    // ID_VAR
    ident(int t, const char *n, int m, int x, int *s, void *f = NULL, int flags = 0)
        : type(t), flags(flags | (m > x ? IDF_READONLY : 0)), name(n), minval(m), maxval(x), fun((identfun)f)
    { storage.i = s; }
    // ID_FVAR
    ident(int t, const char *n, float m, float x, float *s, void *f = NULL, int flags = 0)
        : type(t), flags(flags | (m > x ? IDF_READONLY : 0)), name(n), minvalf(m), maxvalf(x), fun((identfun)f)
    { storage.f = s; }
    // ID_SVAR
    ident(int t, const char *n, char **s, void *f = NULL, int flags = 0)
        : type(t), flags(flags), name(n), fun((identfun)f)
    { storage.s = s; }
    // ID_ALIAS
    ident(int t, const char *n, char *a, int flags)
        : type(t), valtype(VAL_STR), flags(flags), name(n), code(NULL), stack(NULL)
    { val.s = a; }
    ident(int t, const char *n, int a, int flags)
        : type(t), valtype(VAL_INT), flags(flags), name(n), code(NULL), stack(NULL)
    { val.i = a; }
    ident(int t, const char *n, float a, int flags)
        : type(t), valtype(VAL_FLOAT), flags(flags), name(n), code(NULL), stack(NULL)
    { val.f = a; }
    ident(int t, const char *n, int flags)
        : type(t), valtype(VAL_NULL), flags(flags), name(n), code(NULL), stack(NULL)
    {}
    ident(int t, const char *n, const tagval &v, int flags)
        : type(t), valtype(v.type), flags(flags), name(n), code(NULL), stack(NULL)
    { val = v; }
    // ID_COMMAND
    ident(int t, const char *n, const char *args, uint argmask, int numargs, void *f = NULL, int flags = 0)
        : type(t), numargs(numargs), flags(flags), name(n), args(args), argmask(argmask), fun((identfun)f)
    {}

    void changed() { if(fun) fun(); }

    void setval(const tagval &v)
    {
        valtype = v.type;
        val = v;
    }
   
    void setval(const identstack &v)
    {
        valtype = v.valtype;
        val = v.val;
    }
 
    void forcenull()
    {
        if(valtype==VAL_STR) delete[] val.s;
        valtype = VAL_NULL;
    }

    float getfloat() const;
    int getint() const;
    const char *getstr() const;
    void getval(tagval &v) const;
};

static inline bool htcmp(const char *key, const ident &id) { return !strcmp(key, id.name); }

extern void addident(ident *id);

extern tagval *commandret;
extern const char *intstr(int v);
extern void intret(int v);
extern const char *floatstr(float v);
extern void floatret(float v);
extern void stringret(char *s);
extern void result(tagval &v);
extern void result(const char *s);

static inline int parseint(const char *s)
{
    return int(strtol(s, NULL, 0));
}

static inline float parsefloat(const char *s)
{
    // not all platforms (windows) can parse hexadecimal integers via strtod
    char *end;
    double val = strtod(s, &end);
    return val || end==s || (*end!='x' && *end!='X') ? float(val) : float(parseint(s));
}

static inline const char *getstr(const identval &v, int type)
{
    switch(type)
    {
        case VAL_STR: case VAL_MACRO: return v.s;
        case VAL_INT: return intstr(v.i);
        case VAL_FLOAT: return floatstr(v.f);
        default: return "";
    }
}
inline const char *tagval::getstr() const { return ::getstr(*this, type); }
inline const char *ident::getstr() const { return ::getstr(val, valtype); }

static inline int getint(const identval &v, int type)
{
    switch(type)
    {
        case VAL_INT: return v.i;
        case VAL_FLOAT: return int(v.f);
        case VAL_STR: case VAL_MACRO: return parseint(v.s);
        default: return 0;
    }
}
inline int tagval::getint() const { return ::getint(*this, type); }
inline int ident::getint() const { return ::getint(val, valtype); }

static inline float getfloat(const identval &v, int type)
{
    switch(type)
    {
        case VAL_FLOAT: return v.f;
        case VAL_INT: return float(v.i);
        case VAL_STR: case VAL_MACRO: return parsefloat(v.s);
        default: return 0.0f;
    }
}
inline float tagval::getfloat() const { return ::getfloat(*this, type); }
inline float ident::getfloat() const { return ::getfloat(val, valtype); }

inline void ident::getval(tagval &v) const
{
    switch(valtype)
    {
        case VAL_STR: case VAL_MACRO: v.setstr(newstring(val.s)); break;
        case VAL_INT: v.setint(val.i); break;
        case VAL_FLOAT: v.setfloat(val.f); break;
        default: v.setnull(); break;
    }
}

// nasty macros for registering script functions, abuses globals to avoid excessive infrastructure
#define KEYWORD(name, type) static bool __dummy_##name = addkeyword(type, #name)
#define COMMANDN(name, fun, nargs) static bool __dummy_##fun = addcommand(#name, (identfun)fun, nargs)
#define COMMAND(name, nargs) COMMANDN(name, name, nargs)

#define _VAR(name, global, min, cur, max, persist)  int global = variable(#name, min, cur, max, &global, NULL, persist)
#define VARN(name, global, min, cur, max) _VAR(name, global, min, cur, max, 0)
#define VARNP(name, global, min, cur, max) _VAR(name, global, min, cur, max, IDF_PERSIST)
#define VARNR(name, global, min, cur, max) _VAR(name, global, min, cur, max, IDF_OVERRIDE)
#define VAR(name, min, cur, max) _VAR(name, name, min, cur, max, 0)
#define VARP(name, min, cur, max) _VAR(name, name, min, cur, max, IDF_PERSIST)
#define VARR(name, min, cur, max) _VAR(name, name, min, cur, max, IDF_OVERRIDE)
#define _VARF(name, global, min, cur, max, body, persist)  void var_##name(); int global = variable(#name, min, cur, max, &global, var_##name, persist); void var_##name() { body; }
#define VARFN(name, global, min, cur, max, body) _VARF(name, global, min, cur, max, body, 0)
#define VARF(name, min, cur, max, body) _VARF(name, name, min, cur, max, body, 0)
#define VARFP(name, min, cur, max, body) _VARF(name, name, min, cur, max, body, IDF_PERSIST)
#define VARFR(name, min, cur, max, body) _VARF(name, name, min, cur, max, body, IDF_OVERRIDE)

#define _HVAR(name, global, min, cur, max, persist)  int global = variable(#name, min, cur, max, &global, NULL, persist | IDF_HEX)
#define HVARN(name, global, min, cur, max) _HVAR(name, global, min, cur, max, 0)
#define HVARNP(name, global, min, cur, max) _HVAR(name, global, min, cur, max, IDF_PERSIST)
#define HVARNR(name, global, min, cur, max) _HVAR(name, global, min, cur, max, IDF_OVERRIDE)
#define HVAR(name, min, cur, max) _HVAR(name, name, min, cur, max, 0)
#define HVARP(name, min, cur, max) _HVAR(name, name, min, cur, max, IDF_PERSIST)
#define HVARR(name, min, cur, max) _HVAR(name, name, min, cur, max, IDF_OVERRIDE)
#define _HVARF(name, global, min, cur, max, body, persist)  void var_##name(); int global = variable(#name, min, cur, max, &global, var_##name, persist | IDF_HEX); void var_##name() { body; }
#define HVARFN(name, global, min, cur, max, body) _HVARF(name, global, min, cur, max, body, 0)
#define HVARF(name, min, cur, max, body) _HVARF(name, name, min, cur, max, body, 0)
#define HVARFP(name, min, cur, max, body) _HVARF(name, name, min, cur, max, body, IDF_PERSIST)
#define HVARFR(name, min, cur, max, body) _HVARF(name, name, min, cur, max, body, IDF_OVERRIDE)

#define _FVAR(name, global, min, cur, max, persist) float global = fvariable(#name, min, cur, max, &global, NULL, persist)
#define FVARN(name, global, min, cur, max) _FVAR(name, global, min, cur, max, 0)
#define FVARNP(name, global, min, cur, max) _FVAR(name, global, min, cur, max, IDF_PERSIST)
#define FVARNR(name, global, min, cur, max) _FVAR(name, global, min, cur, max, IDF_OVERRIDE)
#define FVAR(name, min, cur, max) _FVAR(name, name, min, cur, max, 0)
#define FVARP(name, min, cur, max) _FVAR(name, name, min, cur, max, IDF_PERSIST)
#define FVARR(name, min, cur, max) _FVAR(name, name, min, cur, max, IDF_OVERRIDE)
#define _FVARF(name, global, min, cur, max, body, persist) void var_##name(); float global = fvariable(#name, min, cur, max, &global, var_##name, persist); void var_##name() { body; }
#define FVARFN(name, global, min, cur, max, body) _FVARF(name, global, min, cur, max, body, 0)
#define FVARF(name, min, cur, max, body) _FVARF(name, name, min, cur, max, body, 0)
#define FVARFP(name, min, cur, max, body) _FVARF(name, name, min, cur, max, body, IDF_PERSIST)
#define FVARFR(name, min, cur, max, body) _FVARF(name, name, min, cur, max, body, IDF_OVERRIDE)

#define _SVAR(name, global, cur, persist) char *global = svariable(#name, cur, &global, NULL, persist)
#define SVARN(name, global, cur) _SVAR(name, global, cur, 0)
#define SVARNP(name, global, cur) _SVAR(name, global, cur, IDF_PERSIST)
#define SVARNR(name, global, cur) _SVAR(name, global, cur, IDF_OVERRIDE)
#define SVAR(name, cur) _SVAR(name, name, cur, 0)
#define SVARP(name, cur) _SVAR(name, name, cur, IDF_PERSIST)
#define SVARR(name, cur) _SVAR(name, name, cur, IDF_OVERRIDE)
#define _SVARF(name, global, cur, body, persist) void var_##name(); char *global = svariable(#name, cur, &global, var_##name, persist); void var_##name() { body; }
#define SVARFN(name, global, cur, body) _SVARF(name, global, cur, body, 0)
#define SVARF(name, cur, body) _SVARF(name, name, cur, body, 0)
#define SVARFP(name, cur, body) _SVARF(name, name, cur, body, IDF_PERSIST)
#define SVARFR(name, cur, body) _SVARF(name, name, cur, body, IDF_OVERRIDE)

// anonymous inline commands, uses nasty template trick with line numbers to keep names unique
#define ICOMMANDNS(name, cmdname, nargs, proto, b) template<int N> struct cmdname; template<> struct cmdname<__LINE__> { static bool init; static void run proto; }; bool cmdname<__LINE__>::init = addcommand(name, (identfun)cmdname<__LINE__>::run, nargs); void cmdname<__LINE__>::run proto \
    { b; }
#define ICOMMANDN(name, cmdname, nargs, proto, b) ICOMMANDNS(#name, cmdname, nargs, proto, b)
#define ICOMMANDNAME(name) _icmd_##name
#define ICOMMAND(name, nargs, proto, b) ICOMMANDN(name, ICOMMANDNAME(name), nargs, proto, b)
#define ICOMMANDSNAME _icmds_
#define ICOMMANDS(name, nargs, proto, b) ICOMMANDNS(name, ICOMMANDSNAME, nargs, proto, b)

enum
{
    CT_PRINT   = 1<<0,
    CT_SPACE   = 1<<1,
    CT_DIGIT   = 1<<2,
    CT_ALPHA   = 1<<3,
    CT_LOWER   = 1<<4,
    CT_UPPER   = 1<<5,
    CT_UNICODE = 1<<6
};

const int cube2unichars[256] =
{
    0, 192, 193, 194, 195, 196, 197, 198, 199, 9, 10, 11, 12, 13, 200, 201,
    202, 203, 204, 205, 206, 207, 209, 210, 211, 212, 213, 214, 216, 217, 218, 219,
    32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
    48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
    64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79,
    80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95,
    96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111,
    112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 220,
    221, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237,
    238, 239, 241, 242, 243, 244, 245, 246, 248, 249, 250, 251, 252, 253, 255, 0x104,
    0x105, 0x106, 0x107, 0x10C, 0x10D, 0x10E, 0x10F, 0x118, 0x119, 0x11A, 0x11B, 0x11E, 0x11F, 0x130, 0x131, 0x141,
    0x142, 0x143, 0x144, 0x147, 0x148, 0x150, 0x151, 0x152, 0x153, 0x158, 0x159, 0x15A, 0x15B, 0x15E, 0x15F, 0x160,
    0x161, 0x164, 0x165, 0x16E, 0x16F, 0x170, 0x171, 0x178, 0x179, 0x17A, 0x17B, 0x17C, 0x17D, 0x17E, 0x404, 0x411,
    0x413, 0x414, 0x416, 0x417, 0x418, 0x419, 0x41B, 0x41F, 0x423, 0x424, 0x426, 0x427, 0x428, 0x429, 0x42A, 0x42B,
    0x42C, 0x42D, 0x42E, 0x42F, 0x431, 0x432, 0x433, 0x434, 0x436, 0x437, 0x438, 0x439, 0x43A, 0x43B, 0x43C, 0x43D,
    0x43F, 0x442, 0x444, 0x446, 0x447, 0x448, 0x449, 0x44A, 0x44B, 0x44C, 0x44D, 0x44E, 0x44F, 0x454, 0x490, 0x491
};

const int uni2cubeoffsets[8] =
{
    0, 256, 658, 658, 512, 658, 658, 658
};
const uchar uni2cubechars[878] =
{
    0, 0, 0, 0, 0, 0, 0, 0, 0, 9, 10, 11, 12, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
    64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95,
    96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    1, 2, 3, 4, 5, 6, 7, 8, 14, 15, 16, 17, 18, 19, 20, 21, 0, 22, 23, 24, 25, 26, 27, 0, 28, 29, 30, 31, 127, 128, 0, 129,
    130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144, 145, 0, 146, 147, 148, 149, 150, 151, 0, 152, 153, 154, 155, 156, 157, 0, 158,
    0, 0, 0, 0, 159, 160, 161, 162, 0, 0, 0, 0, 163, 164, 165, 166, 0, 0, 0, 0, 0, 0, 0, 0, 167, 168, 169, 170, 0, 0, 171, 172,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 173, 174, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 175, 176, 177, 178, 0, 0, 179, 180, 0, 0, 0, 0, 0, 0, 0, 181, 182, 183, 184, 0, 0, 0, 0, 185, 186, 187, 188, 0, 0, 189, 190,
    191, 192, 0, 0, 193, 194, 0, 0, 0, 0, 0, 0, 0, 0, 195, 196, 197, 198, 0, 0, 0, 0, 0, 0, 199, 200, 201, 202, 203, 204, 205, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 17, 0, 0, 206, 83, 73, 21, 74, 0, 0, 0, 0, 0, 0, 0, 65, 207, 66, 208, 209, 69, 210, 211, 212, 213, 75, 214, 77, 72, 79, 215,
    80, 67, 84, 216, 217, 88, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 97, 228, 229, 230, 231, 101, 232, 233, 234, 235, 236, 237, 238, 239, 111, 240,
    112, 99, 241, 121, 242, 120, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 0, 141, 0, 0, 253, 115, 105, 145, 106, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 254, 255, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0
};

const uchar cubelowerchars[256] =
{
    0, 130, 131, 132, 133, 134, 135, 136, 137, 9, 10, 11, 12, 13, 138, 139,
    140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155,
    32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
    48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
    64, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111,
    112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 91, 92, 93, 94, 95,
    96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111,
    112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 156,
    157, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143,
    144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 160,
    160, 162, 162, 164, 164, 166, 166, 168, 168, 170, 170, 172, 172, 105, 174, 176,
    176, 178, 178, 180, 180, 182, 182, 184, 184, 186, 186, 188, 188, 190, 190, 192,
    192, 194, 194, 196, 196, 198, 198, 158, 201, 201, 203, 203, 205, 205, 206, 207,
    208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223,
    224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239,
    240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255
};
const uchar cubeupperchars[256] =
{
    0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
    16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
    32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47,
    48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63,
    64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79,
    80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95,
    96, 65, 66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79,
    80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 123, 124, 125, 126, 127,
    128, 129, 1, 2, 3, 4, 5, 6, 7, 8, 14, 15, 16, 17, 18, 19,
    20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 127, 128, 199, 159,
    159, 161, 161, 163, 163, 165, 165, 167, 167, 169, 169, 171, 171, 173, 73, 175,
    175, 177, 177, 179, 179, 181, 181, 183, 183, 185, 185, 187, 187, 189, 189, 191,
    191, 193, 193, 195, 195, 197, 197, 199, 200, 200, 202, 202, 204, 204, 206, 207,
    208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223,
    224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239,
    240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255
};


#define CUBECTYPE(s, p, d, a, A, u, U) \
    0, U, U, U, U, U, U, U, U, s, s, s, s, s, U, U, \
    U, U, U, U, U, U, U, U, U, U, U, U, U, U, U, U, \
    s, p, p, p, p, p, p, p, p, p, p, p, p, p, p, p, \
    d, d, d, d, d, d, d, d, d, d, p, p, p, p, p, p, \
    p, A, A, A, A, A, A, A, A, A, A, A, A, A, A, A, \
    A, A, A, A, A, A, A, A, A, A, A, p, p, p, p, p, \
    p, a, a, a, a, a, a, a, a, a, a, a, a, a, a, a, \
    a, a, a, a, a, a, a, a, a, a, a, p, p, p, p, U, \
    U, u, u, u, u, u, u, u, u, u, u, u, u, u, u, u, \
    u, u, u, u, u, u, u, u, u, u, u, u, u, u, u, U, \
    u, U, u, U, u, U, u, U, u, U, u, U, u, U, u, U, \
    u, U, u, U, u, U, u, U, u, U, u, U, u, U, u, U, \
    u, U, u, U, u, U, u, U, U, u, U, u, U, u, U, U, \
    U, U, U, U, U, U, U, U, U, U, U, U, U, U, U, U, \
    U, U, U, U, u, u, u, u, u, u, u, u, u, u, u, u, \
    u, u, u, u, u, u, u, u, u, u, u, u, u, u, U, u
const uchar cubectype[256] =
{
    CUBECTYPE(CT_SPACE,
              CT_PRINT,
              CT_PRINT|CT_DIGIT,
              CT_PRINT|CT_ALPHA|CT_LOWER,
              CT_PRINT|CT_ALPHA|CT_UPPER,
              CT_PRINT|CT_UNICODE|CT_ALPHA|CT_LOWER,
              CT_PRINT|CT_UNICODE|CT_ALPHA|CT_UPPER)
};
static inline int iscubeprint(uchar c) { return cubectype[c]&CT_PRINT; }
static inline int iscubespace(uchar c) { return cubectype[c]&CT_SPACE; }
static inline int iscubealpha(uchar c) { return cubectype[c]&CT_ALPHA; }
static inline int iscubealnum(uchar c) { return cubectype[c]&(CT_ALPHA|CT_DIGIT); }
static inline int iscubelower(uchar c) { return cubectype[c]&CT_LOWER; }
static inline int iscubeupper(uchar c) { return cubectype[c]&CT_UPPER; }
static inline int iscubepunct(uchar c) { return cubectype[c] == CT_PRINT; }
static inline int cube2uni(uchar c)
{ 
    return cube2unichars[c]; 
}
static inline uchar uni2cube(int c)
{
    return uint(c) <= 0x7FF ? uni2cubechars[uni2cubeoffsets[c>>8] + (c&0xFF)] : 0;
}
static inline uchar cubelower(uchar c)
{
    return cubelowerchars[c];
}
static inline uchar cubeupper(uchar c)
{
    return cubeupperchars[c];
}

enum
{
    CON_INFO  = 1<<0,
    CON_WARN  = 1<<1,
    CON_ERROR = 1<<2,
    CON_DEBUG = 1<<3,
    CON_INIT  = 1<<4,
    CON_ECHO  = 1<<5,

    CON_FLAGS = 0xFFFF,
    CON_TAG_SHIFT = 16,
    CON_TAG_MASK = (0x7FFF << CON_TAG_SHIFT)
};

extern void conoutf(const char *s, ...);
extern void conoutf(int type, const char *s, ...);
extern void conoutfv(int type, const char *fmt, va_list args);

// from iengine.h
extern int variable(const char *name, int min, int cur, int max, int *storage, identfun fun, int flags);
extern float fvariable(const char *name, float min, float cur, float max, float *storage, identfun fun, int flags);
extern char *svariable(const char *name, const char *cur, char **storage, identfun fun, int flags);
extern void setvar(const char *name, int i, bool dofunc = true, bool doclamp = true);
extern void setfvar(const char *name, float f, bool dofunc = true, bool doclamp = true);
extern void setsvar(const char *name, const char *str, bool dofunc = true);
extern void setvarchecked(ident *id, int val);
extern void setfvarchecked(ident *id, float val);
extern void setsvarchecked(ident *id, const char *val);
extern void touchvar(const char *name);
extern int getvar(const char *name);
extern int getvarmin(const char *name);
extern int getvarmax(const char *name);
extern bool identexists(const char *name);
extern ident *getident(const char *name);
extern ident *newident(const char *name, int flags = 0);
extern ident *readident(const char *name);
extern ident *writeident(const char *name, int flags = 0);
extern bool addcommand(const char *name, identfun fun, const char *narg);
extern bool addkeyword(int type, const char *name);
extern uint *compilecode(const char *p);
extern void keepcode(uint *p);
extern void freecode(uint *p);
extern void executeret(const uint *code, tagval &result = *commandret);
extern void executeret(const char *p, tagval &result = *commandret);
extern void executeret(ident *id, tagval *args, int numargs, bool lookup = false, tagval &result = *commandret);
extern char *executestr(const uint *code);
extern char *executestr(const char *p);
extern char *executestr(ident *id, tagval *args, int numargs, bool lookup = false);
extern char *execidentstr(const char *name, bool lookup = false);
extern int execute(const uint *code);
extern int execute(const char *p);
extern int execute(ident *id, tagval *args, int numargs, bool lookup = false);
extern int execident(const char *name, int noid = 0, bool lookup = false);
extern bool executebool(const uint *code);
extern bool executebool(const char *p);
extern bool executebool(ident *id, tagval *args, int numargs, bool lookup = false);
extern bool execidentbool(const char *name, bool noid = false, bool lookup = false);
extern bool execfile(const char *cfgfile, bool msg = true);
extern void alias(const char *name, const char *action);
extern void alias(const char *name, tagval &v);
extern const char *getalias(const char *name);
extern const char *escapestring(const char *s);
extern const char *escapeid(const char *s);
static inline const char *escapeid(ident &id) { return escapeid(id.name); }
extern bool validateblock(const char *s);
extern void explodelist(const char *s, vector<char *> &elems, int limit = -1);
extern char *indexlist(const char *s, int pos);
extern int listlen(const char *s);
extern void printvar(ident *id);
extern void printvar(ident *id, int i);
extern void printfvar(ident *id, float f);
extern void printsvar(ident *id, const char *s);
extern int clampvar(ident *id, int i, int minval, int maxval);
extern float clampfvar(ident *id, float f, float minval, float maxval);
extern void loopiter(ident *id, identstack &stack, const tagval &v);
extern void loopend(ident *id, identstack &stack);

#define loopstart(id, stack) if((id)->type != ID_ALIAS) return; identstack stack;
static inline void loopiter(ident *id, identstack &stack, int i) { tagval v; v.setint(i); loopiter(id, stack, v); }
static inline void loopiter(ident *id, identstack &stack, float f) { tagval v; v.setfloat(f); loopiter(id, stack, v); }
static inline void loopiter(ident *id, identstack &stack, const char *s) { tagval v; v.setstr(newstring(s)); loopiter(id, stack, v); }
#endif
