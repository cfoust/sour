#include "engine.h"

#define MAXWATERPVS 32

uint numwaterplanes = 0;

static struct
{
    int height;
    vector<materialsurface *> matsurfs;
} waterplanes[MAXWATERPVS];

struct viewcellnode
{
    uchar leafmask;
    union viewcellchild
    {
        int pvs;
        viewcellnode *node;
    } children[8];

    viewcellnode() : leafmask(0xFF)
    {
        loopi(8) children[i].pvs = -1;
    }
    ~viewcellnode()
    {
        loopi(8) if(!(leafmask&(1<<i))) delete children[i].node;
    }
};

viewcellnode *loadviewcells(stream *f)
{
    viewcellnode *p = new viewcellnode;
    p->leafmask = f->getchar();
    loopi(8)
    {
        if(p->leafmask&(1<<i)) p->children[i].pvs = f->getlil<int>();
        else p->children[i].node = loadviewcells(f);
    }
    return p;
}

struct pvsdata
{
    int offset, len;

    pvsdata() {}
    pvsdata(int offset, int len) : offset(offset), len(len) {}
};

static vector<pvsdata> pvs;
static vector<uchar> pvsbuf;

static viewcellnode *viewcells = NULL;

void loadpvs(stream *f, int numpvs)
{
    uint totallen = f->getlil<uint>();
    if(totallen & 0x80000000U)
    {
        totallen &= ~0x80000000U;
        numwaterplanes = f->getlil<uint>();
        loopi(numwaterplanes) waterplanes[i].height = f->getlil<int>();
    }
    int offset = 0;
    loopi(numpvs)
    {
        ushort len = f->getlil<ushort>();
        pvs.add(pvsdata(offset, len));
        offset += len;
    }
    f->read(pvsbuf.reserve(totallen).buf, totallen);
    pvsbuf.advance(totallen);
    viewcells = loadviewcells(f);
}
