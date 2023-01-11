typedef signed char schar;
typedef unsigned char uchar;
typedef unsigned short ushort;
typedef unsigned int uint;
typedef unsigned int GLuint;
typedef unsigned long ulong;
typedef signed long long int llong;
typedef unsigned long long int ullong;

struct surfaceinfo
{
    uchar lmid[2];
    uchar verts, numverts;
};

struct cubeext
{
    vtxarray *va;            // vertex array for children, or NULL
    octaentities *ents;      // map entities inside cube
    surfaceinfo surfaces[6]; // render info for each surface
    int tjoints;             // linked list of t-joints
    uchar maxverts;          // allocated space for verts
};  

struct cube
{
    cube *children;          // points to 8 cube structures which are its children, or NULL. -Z first, then -Y, -X
    cubeext *ext;            // extended info for the cube
    uchar edges[12];     // edges of the cube, each uchar is 2 4bit values denoting the range.
    ushort texture[6];       // one for each face. same order as orient.
    ushort material;         // empty-space material
    uchar merged;            // merged faces of the cube
    uchar escaped;       // mask of which children have escaped merges
};

struct SlotShaderParam
{
    const char *name;
    int loc;
    float val[4];
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
};
