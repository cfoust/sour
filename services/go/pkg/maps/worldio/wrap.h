typedef signed char schar;
typedef unsigned char uchar;
typedef unsigned short ushort;
typedef unsigned int uint;
typedef unsigned int GLuint;
typedef unsigned long ulong;
typedef signed long long int llong;
typedef unsigned long long int ullong;

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
