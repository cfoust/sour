// the interface the game uses to access the engine

extern int curtime;                     // current frame time
extern int lastmillis;                  // last time
extern int elapsedtime;                 // elapsed frame time
extern int totalmillis;                 // total elapsed time
extern uint totalsecs;
extern int gamespeed, paused;

enum
{
    MATF_INDEX_SHIFT  = 0,
    MATF_VOLUME_SHIFT = 2,
    MATF_CLIP_SHIFT   = 5,
    MATF_FLAG_SHIFT   = 8,

    MATF_INDEX  = 3 << MATF_INDEX_SHIFT,
    MATF_VOLUME = 7 << MATF_VOLUME_SHIFT,
    MATF_CLIP   = 7 << MATF_CLIP_SHIFT,
    MATF_FLAGS  = 0xFF << MATF_FLAG_SHIFT
};

enum // cube empty-space materials
{
    MAT_AIR      = 0,                      // the default, fill the empty space with air
    MAT_WATER    = 1 << MATF_VOLUME_SHIFT, // fill with water, showing waves at the surface
    MAT_LAVA     = 2 << MATF_VOLUME_SHIFT, // fill with lava
    MAT_GLASS    = 3 << MATF_VOLUME_SHIFT, // behaves like clip but is blended blueish

    MAT_NOCLIP   = 1 << MATF_CLIP_SHIFT,  // collisions always treat cube as empty
    MAT_CLIP     = 2 << MATF_CLIP_SHIFT,  // collisions always treat cube as solid
    MAT_GAMECLIP = 3 << MATF_CLIP_SHIFT,  // game specific clip material

    MAT_DEATH    = 1 << MATF_FLAG_SHIFT,  // force player suicide
    MAT_ALPHA    = 4 << MATF_FLAG_SHIFT   // alpha blended
};

#define isliquid(mat) ((mat)==MAT_WATER || (mat)==MAT_LAVA)
#define isclipped(mat) ((mat)==MAT_GLASS)
#define isdeadly(mat) ((mat)==MAT_LAVA)

extern void lightent(extentity &e, float height = 8.0f);
extern void lightreaching(const vec &target, vec &color, vec &dir, bool fast = false, extentity *e = 0, float ambient = 0.4f);
extern entity *brightestlight(const vec &target, const vec &dir);

enum { RAY_BB = 1, RAY_POLY = 3, RAY_ALPHAPOLY = 7, RAY_ENTS = 9, RAY_CLIPMAT = 16, RAY_SKIPFIRST = 32, RAY_EDITMAT = 64, RAY_SHADOW = 128, RAY_PASS = 256, RAY_SKIPSKY = 512, RAY_SKYTEX = 1024 };

extern float raycube   (const vec &o, const vec &ray,     float radius = 0, int mode = RAY_CLIPMAT, int size = 0, extentity *t = 0);
extern float raycubepos(const vec &o, const vec &ray, vec &hit, float radius = 0, int mode = RAY_CLIPMAT, int size = 0);
extern float rayfloor  (const vec &o, vec &floor, int mode = 0, float radius = 0);
extern bool  raycubelos(const vec &o, const vec &dest, vec &hitpos);

extern int thirdperson;
extern bool isthirdperson();

extern bool settexture(const char *name, int clamp = 0);

// octaedit

enum { EDIT_FACE = 0, EDIT_TEX, EDIT_MAT, EDIT_FLIP, EDIT_COPY, EDIT_PASTE, EDIT_ROTATE, EDIT_REPLACE, EDIT_DELCUBE, EDIT_REMIP, EDIT_VSLOT, EDIT_UNDO, EDIT_REDO };

struct selinfo
{
    int corner;
    int cx, cxs, cy, cys;
    ivec o, s;
    int grid, orient;
    selinfo() : corner(0), cx(0), cxs(0), cy(0), cys(0), o(0, 0, 0), s(0, 0, 0), grid(8), orient(0) {}
    int size() const    { return s.x*s.y*s.z; }
    int us(int d) const { return s[d]*grid; }
    bool operator==(const selinfo &sel) const { return o==sel.o && s==sel.s && grid==sel.grid && orient==sel.orient; }
    bool validate()
    {
        extern int worldsize;
        if(grid <= 0 || grid >= worldsize) return false;
        if(o.x >= worldsize || o.y >= worldsize || o.z >= worldsize) return false;
        if(o.x < 0) { s.x -= (grid - 1 - o.x)/grid; o.x = 0; } 
        if(o.y < 0) { s.y -= (grid - 1 - o.y)/grid; o.y = 0; } 
        if(o.z < 0) { s.z -= (grid - 1 - o.z)/grid; o.z = 0; } 
        s.x = clamp(s.x, 0, (worldsize - o.x)/grid);
        s.y = clamp(s.y, 0, (worldsize - o.y)/grid);
        s.z = clamp(s.z, 0, (worldsize - o.z)/grid);
        return s.x > 0 && s.y > 0 && s.z > 0;
    }
};

struct editinfo;
extern editinfo *localedit;

extern bool editmode;

extern int shouldpacktex(int index);
extern bool packeditinfo(editinfo *e, int &inlen, uchar *&outbuf, int &outlen);
extern bool unpackeditinfo(editinfo *&e, const uchar *inbuf, int inlen, int outlen);
extern void freeeditinfo(editinfo *&e);
extern void pruneundos(int maxremain = 0);
extern bool packundo(int op, int &inlen, uchar *&outbuf, int &outlen);
extern bool unpackundo(const uchar *inbuf, int inlen, int outlen);
extern bool noedit(bool view = false, bool msg = true);
extern void toggleedit(bool force = true);
extern void mpeditface(int dir, int mode, selinfo &sel, bool local);
extern void mpedittex(int tex, int allfaces, selinfo &sel, bool local);
extern bool mpedittex(int tex, int allfaces, selinfo &sel, ucharbuf &buf);
extern void mpeditmat(int matid, int filter, selinfo &sel, bool local);
extern void mpflip(selinfo &sel, bool local);
extern void mpcopy(editinfo *&e, selinfo &sel, bool local);
extern void mppaste(editinfo *&e, selinfo &sel, bool local);
extern void mprotate(int cw, selinfo &sel, bool local);
extern void mpreplacetex(int oldtex, int newtex, bool insel, selinfo &sel, bool local);
extern bool mpreplacetex(int oldtex, int newtex, bool insel, selinfo &sel, ucharbuf &buf);
extern void mpdelcube(selinfo &sel, bool local);
extern bool mpeditvslot(int delta, int allfaces, selinfo &sel, ucharbuf &buf);
extern void mpremip(bool local);

// texture

struct VSlot;

extern void packvslot(vector<uchar> &buf, int index);
extern void packvslot(vector<uchar> &buf, const VSlot *vs);

// command
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

// console

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

extern void conoutf(const char *s, ...) PRINTFARGS(1, 2);
extern void conoutf(int type, const char *s, ...) PRINTFARGS(2, 3);
extern void conoutf(int type, int tag, const char *s, ...) PRINTFARGS(3, 4);
extern void conoutfv(int type, const char *fmt, va_list args);

extern FILE *getlogfile();
extern void setlogfile(const char *fname);
extern void closelogfile();
extern void logoutfv(const char *fmt, va_list args);
extern void logoutf(const char *fmt, ...) PRINTFARGS(1, 2);

// menus
extern vec menuinfrontofplayer();
extern void newgui(char *name, char *contents, char *header = NULL, char *init = NULL);
extern void showgui(const char *name);
extern int cleargui(int n = 0);

// octa
extern int lookupmaterial(const vec &o);

static inline bool insideworld(const vec &o)
{
	extern int worldsize;
    return o.x>=0 && o.x<worldsize && o.y>=0 && o.y<worldsize && o.z>=0 && o.z<worldsize;
}

static inline bool insideworld(const ivec &o)
{
	extern int worldsize;
    return uint(o.x)<uint(worldsize) && uint(o.y)<uint(worldsize) && uint(o.z)<uint(worldsize);
}

// world
extern bool emptymap(int factor, bool force, const char *mname = "", bool usecfg = true);
extern bool enlargemap(bool force);
extern int findentity(int type, int index = 0, int attr1 = -1, int attr2 = -1);
extern void findents(int low, int high, bool notspawned, const vec &pos, const vec &radius, vector<int> &found);
extern void mpeditent(int i, const vec &o, int type, int attr1, int attr2, int attr3, int attr4, int attr5, bool local);
extern vec getselpos();
extern int getworldsize();
extern int getmapversion();
extern void renderentcone(const extentity &e, const vec &dir, float radius, float angle);
extern void renderentarrow(const extentity &e, const vec &dir, float radius);
extern void renderentattachment(const extentity &e);
extern void renderentsphere(const extentity &e, float radius);
extern void renderentring(const extentity &e, float radius, int axis = 0);

// main
extern void fatal(const char *s, ...) PRINTFARGS(1, 2);

// rendertext
extern bool setfont(const char *name);
extern void pushfont();
extern bool popfont();
extern void gettextres(int &w, int &h);
extern void draw_text(const char *str, int left, int top, int r = 255, int g = 255, int b = 255, int a = 255, int cursor = -1, int maxwidth = -1);
extern void draw_textf(const char *fstr, int left, int top, ...) PRINTFARGS(1, 4);
extern float text_widthf(const char *str);
extern void text_boundsf(const char *str, float &width, float &height, int maxwidth = -1);
extern int text_visible(const char *str, float hitx, float hity, int maxwidth);
extern void text_posf(const char *str, int cursor, float &cx, float &cy, int maxwidth);

static inline int text_width(const char *str)
{
    return int(ceil(text_widthf(str)));
}

static inline void text_bounds(const char *str, int &width, int &height, int maxwidth = -1)
{
    float widthf, heightf;
    text_boundsf(str, widthf, heightf, maxwidth);
    width = int(ceil(widthf));
    height = int(ceil(heightf));
}

static inline void text_pos(const char *str, int cursor, int &cx, int &cy, int maxwidth)
{
    float cxf, cyf;
    text_posf(str, cursor, cxf, cyf, maxwidth);
    cx = int(cxf);
    cy = int(cyf);
}

// renderva
enum
{
    DL_SHRINK = 1<<0,
    DL_EXPAND = 1<<1,
    DL_FLASH  = 1<<2
};

extern void adddynlight(const vec &o, float radius, const vec &color, int fade = 0, int peak = 0, int flags = 0, float initradius = 0, const vec &initcolor = vec(0, 0, 0), physent *owner = NULL);
extern void dynlightreaching(const vec &target, vec &color, vec &dir, bool hud = false);
extern void removetrackeddynlights(physent *owner = NULL);

// rendergl
extern physent *camera1;
extern vec worldpos, camdir, camright, camup;

// renderparticles
enum
{
    PART_BLOOD = 0,
    PART_WATER,
    PART_SMOKE,
    PART_STEAM,
    PART_FLAME,
    PART_FIREBALL1, PART_FIREBALL2, PART_FIREBALL3,
    PART_STREAK, PART_LIGHTNING,
    PART_EXPLOSION, PART_EXPLOSION_BLUE,
    PART_SPARK, PART_EDIT,
    PART_SNOW,
    PART_MUZZLE_FLASH1, PART_MUZZLE_FLASH2, PART_MUZZLE_FLASH3,
    PART_HUD_ICON,
    PART_HUD_ICON_GREY,
    PART_TEXT,
    PART_TEXT_ICON,
    PART_METER, PART_METER_VS,
    PART_LENS_FLARE
};

// decal
enum
{
    DECAL_SCORCH = 0,
    DECAL_BLOOD,
    DECAL_BULLET
};

extern void adddecal(int type, const vec &center, const vec &surface, float radius, const bvec &color = bvec(0xFF, 0xFF, 0xFF), int info = 0);

// worldio
extern bool load_world(const char *mname, const char *cname = NULL);
extern bool save_world(const char *mname, bool nolms = false);
extern void fixmapname(char *name);
extern void getmapfilenames(const char *fname, const char *cname, char *pakname, char *mapname, char *cfgname);
extern uint getmapcrc();
extern void clearmapcrc();
extern bool loadents(const char *fname, vector<entity> &ents, uint *crc = NULL);

// physics
extern vec collidewall;
extern int collideinside;
extern physent *collideplayer;

extern void moveplayer(physent *pl, int moveres, bool local);
extern bool moveplayer(physent *pl, int moveres, bool local, int curtime);
extern bool collide(physent *d, const vec &dir = vec(0, 0, 0), float cutoff = 0.0f, bool playercol = true, bool insideplayercol = false);
extern bool bounce(physent *d, float secs, float elasticity, float waterfric, float grav);
extern bool bounce(physent *d, float elasticity, float waterfric, float grav);
extern void avoidcollision(physent *d, const vec &dir, physent *obstacle, float space);
extern bool overlapsdynent(const vec &o, float radius);
extern bool movecamera(physent *pl, const vec &dir, float dist, float stepdist);
extern void physicsframe();
extern void dropenttofloor(entity *e);
extern bool droptofloor(vec &o, float radius, float height);

extern void vecfromyawpitch(float yaw, float pitch, int move, int strafe, vec &m);
extern void vectoyawpitch(const vec &v, float &yaw, float &pitch);
extern bool moveplatform(physent *p, const vec &dir);
extern void updatephysstate(physent *d);
extern void cleardynentcache();
extern void updatedynentcache(physent *d);
extern bool entinmap(dynent *d, bool avoidplayers = false);
extern void findplayerspawn(dynent *d, int forceent = -1, int tag = 0);
