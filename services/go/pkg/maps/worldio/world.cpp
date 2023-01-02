// world.cpp: core map management stuff

#include "engine.h"

int mapversion = MAPVERSION;
int worldscale = 0;
int worldsize = 0;
//SVARR(maptitle, "Untitled Map by Unknown");

int octaentsize = 64;
int entselradius = 2;


extern selinfo sel;
extern bool havesel;
int entlooplevel = 0;
int efocus = -1, enthover = -1, entorient = -1, oldhover = -1;
bool undonext = true;

int entediting = 0;

bool pointinsel(const selinfo &sel, const vec &o)
{
    return(o.x <= sel.o.x+sel.s.x*sel.grid
        && o.x >= sel.o.x
        && o.y <= sel.o.y+sel.s.y*sel.grid
        && o.y >= sel.o.y
        && o.z <= sel.o.z+sel.s.z*sel.grid
        && o.z >= sel.o.z);
}

vector<int> entgroup;

bool haveselent()
{
    return entgroup.length() > 0;
}

void entcancel()
{
    entgroup.shrink(0);
}

void entadd(int id)
{
    undonext = true;
    entgroup.add(id);
}

int attachradius = 100;

// convenience macros implicitly define:
// e         entity, currently edited ent
// n         int,    index to currently edited ent
#define addimplicit(f)    { if(entgroup.empty() && enthover>=0) { entadd(enthover); undonext = (enthover != oldhover); f; entgroup.drop(); } else f; }
#define entfocusv(i, f, v){ int n = efocus = (i); if(n>=0) { extentity &e = *v[n]; f; } }
#define entfocus(i, f)    entfocusv(i, f, entities::getents())
#define enteditv(i, f, v) \
{ \
    entfocusv(i, \
    { \
        int oldtype = e.type; \
        removeentity(n);  \
        f; \
        if(oldtype!=e.type) detachentity(e); \
        if(e.type!=ET_EMPTY) { addentity(n); if(oldtype!=e.type) attachentity(e); } \
        entities::editent(n, true); \
    }, v); \
}
#define entedit(i, f)   enteditv(i, f, entities::getents())
#define addgroup(exp)   { vector<extentity *> &ents = entities::getents(); loopv(ents) entfocusv(i, if(exp) entadd(n), ents); }
#define setgroup(exp)   { entcancel(); addgroup(exp); }
#define groupeditloop(f){ vector<extentity *> &ents = entities::getents(); entlooplevel++; int _ = efocus; loopv(entgroup) enteditv(entgroup[i], f, ents); efocus = _; entlooplevel--; }
#define groupeditpure(f){ if(entlooplevel>0) { entedit(efocus, f); } else groupeditloop(f); }
#define groupeditundo(f){ makeundoent(); groupeditpure(f); }
#define groupedit(f)    { addimplicit(groupeditundo(f)); }

int entselsnap = 0;
int entmovingshadow = 1;

extern void boxs(int orient, vec o, const vec &s, float size);
extern void boxs(int orient, vec o, const vec &s);
extern void boxs3D(const vec &o, vec s, int g);
extern bool editmoveplane(const vec &o, const vec &ray, int d, float off, vec &handle, vec &dest, bool first);

int entmoving = 0;

int showentradius = 1;

bool enttoggle(int id)
{
    undonext = true;
    int i = entgroup.find(id);
    if(i < 0)
        entadd(id);
    else
        entgroup.remove(i);
    return i < 0;
}

int entitysurf = 0;

int entdrop = 2;

int entcamdir = 1;

static int keepents = 0;

struct spawninfo { const extentity *e; float weight; };

void splitocta(cube *c, int size)
{
    if(size <= 0x1000) return;
    loopi(8)
    {
        if(!c[i].children) c[i].children = newcubes(isempty(c[i]) ? F_EMPTY : F_SOLID);
        splitocta(c[i].children, size>>1);
    }
}

bool emptymap(int scale)    // main empty world creation routine
{
    worldroot = newcubes(F_EMPTY);
    loopi(4) solidfaces(worldroot[i]);

    if(worldsize > 0x1000) splitocta(worldroot, worldsize>>1);

    return true;
}

static bool isallempty(cube &c)
{
    if(!c.children) return isempty(c);
    loopi(8) if(!isallempty(c.children[i])) return false;
    return true;
}


int getworldsize() { return worldsize; }
int getmapversion() { return mapversion; }

