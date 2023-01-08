// world.cpp: core map management stuff

#include "engine.h"

int mapversion = MAPVERSION;
int worldscale = 0;
int worldsize = 0;
//SVARR(maptitle, "Untitled Map by Unknown");

int octaentsize = 64;
int entselradius = 2;

char *entname(entity &e)
{
    static string fullentname;
    copystring(fullentname, entities::entname(e.type));
    const char *einfo = entities::entnameinfo(e);
    if(*einfo)
    {
        concatstring(fullentname, ": ");
        concatstring(fullentname, einfo);
    }
    return fullentname;
}

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

undoblock *newundoent()
{
    int numents = entgroup.length();
    if(numents <= 0) return NULL;
    undoblock *u = (undoblock *)new uchar[sizeof(undoblock) + numents*sizeof(undoent)];
    u->numents = numents;
    undoent *e = (undoent *)(u + 1);
    loopv(entgroup)
    {
        e->i = entgroup[i];
        e->e = *entities::getents()[entgroup[i]];
        e++;
    }
    return u;
}

void makeundoent()
{
    if(!undonext) return;
    undonext = false;
    oldhover = enthover;
    undoblock *u = newundoent();
    if(u) addundo(u);
}

void detachentity(extentity &e)
{
    if(!e.attached) return;
    e.attached->attached = NULL;
    e.attached = NULL;
}

int attachradius = 100;

void attachentity(extentity &e)
{
    switch(e.type)
    {
        case ET_SPOTLIGHT:
            break;

        default:
            if(e.type<ET_GAMESPECIFIC || !entities::mayattach(e)) return;
            break;
    }

    detachentity(e);

    vector<extentity *> &ents = entities::getents();
    int closest = -1;
    float closedist = 1e10f;
    loopv(ents)
    {
        extentity *a = ents[i];
        if(a->attached) continue;
        switch(e.type)
        {
            case ET_SPOTLIGHT: 
                if(a->type!=ET_LIGHT) continue; 
                break;

            default:
                if(e.type<ET_GAMESPECIFIC || !entities::attachent(e, *a)) continue;
                break;
        }
        float dist = e.o.dist(a->o);
        if(dist < closedist)
        {
            closest = i;
            closedist = dist;
        }
    }
    if(closedist>attachradius) return;
    e.attached = ents[closest];
    ents[closest]->attached = &e;
}

void attachentities()
{
    vector<extentity *> &ents = entities::getents();
    loopv(ents) attachentity(*ents[i]);
}

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

vec getselpos()
{
    vector<extentity *> &ents = entities::getents();
    if(entgroup.length() && ents.inrange(entgroup[0])) return ents[entgroup[0]]->o;
    if(ents.inrange(enthover)) return ents[enthover]->o;
    return vec(sel.o);
}

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

extentity *newentity(bool local, const vec &o, int type, int v1, int v2, int v3, int v4, int v5, int &idx)
{
    vector<extentity *> &ents = entities::getents();
    if(local)
    {
        idx = -1;
        for(int i = keepents; i < ents.length(); i++) if(ents[i]->type == ET_EMPTY) { idx = i; break; }
        if(idx < 0 && ents.length() >= MAXENTS) { conoutf(CON_ERROR, "too many entities"); return NULL; }
    }
    else while(ents.length() < idx) ents.add(entities::newentity())->type = ET_EMPTY;
    extentity &e = *entities::newentity();
    e.o = o;
    e.attr1 = v1;
    e.attr2 = v2;
    e.attr3 = v3;
    e.attr4 = v4;
    e.attr5 = v5;
    e.type = type;
    e.reserved = 0;
    e.light.color = vec(1, 1, 1);
    e.light.dir = vec(0, 0, 1);
    if(local)
    {
        if(entcamdir) switch(type)
        {
            case ET_MAPMODEL:
            case ET_PLAYERSTART:
                e.attr5 = e.attr4;
                e.attr4 = e.attr3;
                e.attr3 = e.attr2;
                e.attr2 = e.attr1;
                e.attr1 = (int)camera1->yaw;
                break;
        }
        entities::fixentity(e);
    }
    if(ents.inrange(idx)) { entities::deleteentity(ents[idx]); ents[idx] = &e; }
    else { idx = ents.length(); ents.add(&e); }
    return &e;
}

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

void mpeditent(int i, const vec &o, int type, int attr1, int attr2, int attr3, int attr4, int attr5, bool local)
{
    if(i < 0 || i >= MAXENTS) return;
    vector<extentity *> &ents = entities::getents();
    if(ents.length()<=i)
    {
        extentity *e = newentity(local, o, type, attr1, attr2, attr3, attr4, attr5, i);
        if(!e) return;
    }
    else
    {
        extentity &e = *ents[i];
        int oldtype = e.type;
        e.type = type;
        e.o = o;
        e.attr1 = attr1; e.attr2 = attr2; e.attr3 = attr3; e.attr4 = attr4; e.attr5 = attr5;
    }
    entities::editent(i, local);
}
