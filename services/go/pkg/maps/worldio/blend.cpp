#include "engine.h"

enum
{
    BM_BRANCH = 0,
    BM_SOLID,
    BM_IMAGE
};

struct BlendMapBranch;
struct BlendMapSolid;
struct BlendMapImage;

struct BlendMapNode
{
    union
    {
        BlendMapBranch *branch;
        BlendMapSolid *solid;
        BlendMapImage *image;
    };

    void cleanup(int type);
    void splitsolid(uchar &type, uchar val);
};

struct BlendMapBranch
{
    uchar type[4];
    BlendMapNode children[4];

    ~BlendMapBranch()
    {
        loopi(4) children[i].cleanup(type[i]);
    }

    uchar shrink(BlendMapNode &child, int quadrant);
};

struct BlendMapSolid
{
    uchar val;

    BlendMapSolid(uchar val) : val(val) {}
};

#define BM_SCALE 1
#define BM_IMAGE_SIZE 64

struct BlendMapImage
{
    uchar data[BM_IMAGE_SIZE*BM_IMAGE_SIZE];
};

void BlendMapNode::cleanup(int type)
{
    switch(type)
    {
        case BM_BRANCH: delete branch; break;
        case BM_IMAGE: delete image; break;
    }
}

#define DEFBMSOLIDS(n) n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12, n+13, n+14, n+15

static BlendMapSolid bmsolids[256] = 
{
    DEFBMSOLIDS(0x00), DEFBMSOLIDS(0x10), DEFBMSOLIDS(0x20), DEFBMSOLIDS(0x30),
    DEFBMSOLIDS(0x40), DEFBMSOLIDS(0x50), DEFBMSOLIDS(0x60), DEFBMSOLIDS(0x70),
    DEFBMSOLIDS(0x80), DEFBMSOLIDS(0x90), DEFBMSOLIDS(0xA0), DEFBMSOLIDS(0xB0),
    DEFBMSOLIDS(0xC0), DEFBMSOLIDS(0xD0), DEFBMSOLIDS(0xE0), DEFBMSOLIDS(0xF0),
};

struct BlendMapRoot : BlendMapNode
{
    uchar type;

    BlendMapRoot() : type(BM_SOLID) { solid = &bmsolids[0xFF]; }
    BlendMapRoot(uchar type, const BlendMapNode &node) : BlendMapNode(node), type(type) {}

    void cleanup() { BlendMapNode::cleanup(type); }

    void shrink(int quadrant)
    {
        if(type == BM_BRANCH) 
        {
            BlendMapRoot oldroot = *this;
            type = branch->shrink(*this, quadrant);
            oldroot.cleanup();
        }
    }
};

static BlendMapRoot blendmap;

bool loadblendmap(stream *f, uchar &type, BlendMapNode &node)
{
    type = f->getchar();
    switch(type)
    {
        case BM_SOLID:
        {
            int val = f->getchar();
            if(val<0 || val>0xFF) return false;
            node.solid = &bmsolids[val];
            break;
        }

        case BM_IMAGE:
            node.image = new BlendMapImage;
            if(f->read(node.image->data, sizeof(node.image->data)) != sizeof(node.image->data))
                return false;
            break;

        case BM_BRANCH:
            node.branch = new BlendMapBranch;
            loopi(4) { node.branch->type[i] = BM_SOLID; node.branch->children[i].solid = &bmsolids[0xFF]; }
            loopi(4) if(!loadblendmap(f, node.branch->type[i], node.branch->children[i]))
                return false;
            break;

        default:
            type = BM_SOLID;
            node.solid = &bmsolids[0xFF];
            return false;
    }
    return true;
}

void resetblendmap()
{
    blendmap.cleanup();
    blendmap.type = BM_SOLID;
    blendmap.solid = &bmsolids[0xFF];
}

bool loadblendmap(stream *f, int info)
{
    resetblendmap();
    return loadblendmap(f, blendmap.type, blendmap);
}
