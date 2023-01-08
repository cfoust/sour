struct SlotShaderParam
{
    const char *name;
    int loc;
    float val[4];
};

enum 
{ 
    SHADER_DEFAULT    = 0, 
    SHADER_NORMALSLMS = 1<<0, 
    SHADER_ENVMAP     = 1<<1,
    SHADER_OPTION     = 1<<3,

    SHADER_INVALID    = 1<<8,
    SHADER_DEFERRED   = 1<<9
};

#define MAXSHADERDETAIL 3
#define MAXVARIANTROWS 5

extern int shaderdetail;

struct Slot;
struct VSlot;

struct UniformLoc
{
    const char *name, *blockname;
    int loc, version, binding, stride, offset, size;
    void *data;
    UniformLoc(const char *name = NULL, const char *blockname = NULL, int binding = -1, int stride = -1) : name(name), blockname(blockname), loc(-1), version(-1), binding(binding), stride(stride), offset(-1), size(-1), data(NULL) {}
};

struct AttribLoc
{
    const char *name;
    int loc;
    AttribLoc(const char *name = NULL, int loc = -1) : name(name), loc(loc) {}
};

#define SETSHADER(name, ...) \
    do { \
        static Shader *name##shader = NULL; \
        if(!name##shader) name##shader = lookupshaderbyname(#name); \
        name##shader->set(__VA_ARGS__); \
    } while(0)
#define SETVARIANT(name, ...) \
    do { \
        static Shader *name##shader = NULL; \
        if(!name##shader) name##shader = lookupshaderbyname(#name); \
        name##shader->setvariant(__VA_ARGS__); \
    } while(0)

// management of texture slots
// each texture slot can have multiple texture frames, of which currently only the first is used
// additional frames can be used for various shaders

struct Texture
{
    enum
    {
        IMAGE      = 0,
        CUBEMAP    = 1,
        TYPE       = 0xFF,
        
        STUB       = 1<<8,
        TRANSIENT  = 1<<9,
        COMPRESSED = 1<<10, 
        ALPHA      = 1<<11,
        MIRROR     = 1<<12,
        FLAGS      = 0xFF00
    };

    char *name;
    int type, w, h, xs, ys, bpp, clamp;
    bool mipmap, canreduce;

    Texture() {}
};

enum
{
    TEX_DIFFUSE = 0,
    TEX_UNKNOWN,
    TEX_DECAL,
    TEX_NORMAL,
    TEX_GLOW,
    TEX_SPEC,
    TEX_DEPTH,
    TEX_ALPHA,
    TEX_ENVMAP
};

enum 
{ 
    VSLOT_SHPARAM = 0, 
    VSLOT_SCALE, 
    VSLOT_ROTATION, 
    VSLOT_OFFSET, 
    VSLOT_SCROLL, 
    VSLOT_LAYER, 
    VSLOT_ALPHA,
    VSLOT_COLOR,
    VSLOT_NUM 
};
   
struct VSlot
{
    Slot *slot;
    VSlot *next;
    int index, changed;
    vector<SlotShaderParam> params;
    bool linked;
    float scale;
    int rotation;
    ivec2 offset;
    vec2 scroll;
    int layer;
    float alphafront, alphaback;
    vec colorscale;
    vec glowcolor;

    VSlot(Slot *slot = NULL, int index = -1) : slot(slot), next(NULL), index(index), changed(0)
    { 
        reset();
        if(slot) addvariant(slot); 
    }

    void addvariant(Slot *slot);

    void reset()
    {
        linked = false;
        scale = 1;
        rotation = 0;
        offset = ivec2(0, 0);
        scroll = vec2(0, 0);
        layer = 0;
        alphafront = 0.5f;
        alphaback = 0;
        colorscale = vec(1, 1, 1);
        glowcolor = vec(1, 1, 1);
    }

    void cleanup()
    {
        linked = false;
    }
};

struct Slot
{
    struct Tex
    {
        int type;
        Texture *t;
        string name;
        int combined;
    };

    int index;
    vector<Tex> sts;
    vector<SlotShaderParam> params;
    VSlot *variants;
    bool loaded;
    uint texmask;
    char *autograss;
    Texture *grasstex, *thumbnail;
    char *layermaskname;
    int layermaskmode;
    float layermaskscale;

    Slot(int index = -1) : index(index), variants(NULL), autograss(NULL), layermaskname(NULL) { reset(); }
    
    void reset()
    {
        sts.shrink(0);
        params.shrink(0);
        loaded = false;
        texmask = 0;
        DELETEA(autograss);
        grasstex = NULL;
        thumbnail = NULL;
        DELETEA(layermaskname);
        layermaskmode = 0;
        layermaskscale = 1;
    }

    void cleanup()
    {
        loaded = false;
        grasstex = NULL;
        thumbnail = NULL;
        loopv(sts) 
        {
            Tex &t = sts[i];
            t.t = NULL;
            t.combined = -1;
        }
    }
};

inline void VSlot::addvariant(Slot *slot)
{
    if(!slot->variants) slot->variants = this;
    else
    {
        VSlot *prev = slot->variants;
        while(prev->next) prev = prev->next;
        prev->next = this;
    }
}

struct MSlot : Slot, VSlot
{
    MSlot() : VSlot(this) {}

    void reset()
    {
        Slot::reset();
        VSlot::reset();
    }

    void cleanup()
    {
        Slot::cleanup();
        VSlot::cleanup();
    }
};

struct texrotation
{
    bool flipx, flipy, swapxy;
};

extern const texrotation texrotations[8];
extern Texture *notexture;
extern int reservevpparams, maxvsuniforms, maxfsuniforms;

extern Texture *loadthumbnail(Slot &slot);
extern void resetslotshader();
extern void setslotshader(Slot &s);
extern void linkslotshader(Slot &s, bool load = true);
extern void linkvslotshader(VSlot &s, bool load = true);
extern void linkslotshaders();
extern const char *getshaderparamname(const char *name, bool insert = true);
extern void setupshaders();
extern void reloadshaders();
extern void cleanupshaders();

#define MAXDYNLIGHTS 5
#define DYNLIGHTBITS 6
#define DYNLIGHTMASK ((1<<DYNLIGHTBITS)-1)

#define MAXBLURRADIUS 7

extern void setupblurkernel(int radius, float sigma, float *weights, float *offsets);
extern void setblurshader(int pass, int size, int radius, float *weights, float *offsets);

extern MSlot &lookupmaterialslot(int slot, bool load = true);
extern Slot &lookupslot(int slot, bool load = true);
extern VSlot &lookupvslot(int slot, bool load = true);
extern VSlot *findvslot(Slot &slot, const VSlot &src, const VSlot &delta);
extern VSlot *editvslot(const VSlot &src, const VSlot &delta);
extern void mergevslot(VSlot &dst, const VSlot &src, const VSlot &delta);
extern void packvslot(vector<uchar> &buf, const VSlot &src);
extern bool unpackvslot(ucharbuf &buf, VSlot &dst, bool delta);

extern Slot dummyslot;
extern VSlot dummyvslot;
extern vector<Slot *> slots;
extern vector<VSlot *> vslots;
