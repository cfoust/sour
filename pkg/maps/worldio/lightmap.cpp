#include "engine.h"
#include "texture.h"

vector<LightMap> lightmaps;

static void rotatenormals(LightMap &lmlv, int x, int y, int w, int h, bool flipx, bool flipy, bool swapxy)
{
    uchar *lv = lmlv.data + 3*(y*LM_PACKW + x);
    int stride = 3*(LM_PACKW-w);
    loopi(h)
    {
        loopj(w)
        {
            if(flipx) lv[0] = 255 - lv[0];
            if(flipy) lv[1] = 255 - lv[1];
            if(swapxy) swap(lv[0], lv[1]);
            lv += 3;
        }
        lv += stride;
    }
}

static void rotatenormals(cube *c)
{
    loopi(8)
    {
        cube &ch = c[i];
        if(ch.children)
        {
            rotatenormals(ch.children);
            continue;
        }
        else if(!ch.ext) continue;
        loopj(6) if(lightmaps.inrange(ch.ext->surfaces[j].lmid[0]+1-LMID_RESERVED))
        {
            VSlot &vslot = lookupvslot(ch.texture[j], false);
            if(!vslot.rotation) continue;
            surfaceinfo &surface = ch.ext->surfaces[j];
            int numverts = surface.numverts&MAXFACEVERTS;
            if(!numverts) continue;
            LightMap &lmlv = lightmaps[surface.lmid[0]+1-LMID_RESERVED];
            if((lmlv.type&LM_TYPE)!=LM_BUMPMAP1) continue;
            ushort x1 = USHRT_MAX, y1 = USHRT_MAX, x2 = 0, y2 = 0;
            vertinfo *verts = ch.ext->verts() + surface.verts;
            loopk(numverts)
            {
                vertinfo &v = verts[k];
                x1 = min(x1, v.u);
                y1 = min(y1, v.u);
                x2 = max(x2, v.u);
                y2 = max(y2, v.v);
            }
            if(x1 > x2 || y1 > y2) continue;
            x1 /= (USHRT_MAX+1)/LM_PACKW;
            y1 /= (USHRT_MAX+1)/LM_PACKH;
            x2 /= (USHRT_MAX+1)/LM_PACKW;
            y2 /= (USHRT_MAX+1)/LM_PACKH;
            const texrotation &r = texrotations[vslot.rotation < 4 ? 4-vslot.rotation : vslot.rotation];
            rotatenormals(lmlv, x1, y1, x2-x1, y1-y1, r.flipx, r.flipy, r.swapxy);
        }
    }
}

void fixlightmapnormals()
{
    rotatenormals(worldroot);
}

void fixrotatedlightmaps(cube &c, const ivec &co, int size)
{
    if(c.children)
    {
        loopi(8) fixrotatedlightmaps(c.children[i], ivec(i, co, size>>1), size>>1);
        return;
    }
    if(!c.ext) return;
    loopi(6) 
    {
        if(c.merged&(1<<i)) continue;
        surfaceinfo &surf = c.ext->surfaces[i];
        int numverts = surf.numverts&MAXFACEVERTS;
        if(numverts!=4 || (surf.lmid[0] < LMID_RESERVED && surf.lmid[1] < LMID_RESERVED)) continue;
        vertinfo *verts = c.ext->verts() + surf.verts;
        int vis = visibletris(c, i, co, size);
        if(!vis || vis==3) continue;
        if((verts[0].u != verts[1].u || verts[0].v != verts[1].v) &&
           (verts[0].u != verts[3].u || verts[0].v != verts[3].v) &&
           (verts[2].u != verts[1].u || verts[2].v != verts[1].v) &&
           (verts[2].u != verts[3].u || verts[2].v != verts[3].v))
            continue;
        if(vis&4)
        {
            vertinfo tmp = verts[0];
            verts[0].x = verts[1].x; verts[0].y = verts[1].y; verts[0].z = verts[1].z;
            verts[1].x = verts[2].x; verts[1].y = verts[2].y; verts[1].z = verts[2].z;
            verts[2].x = verts[3].x; verts[2].y = verts[3].y; verts[2].z = verts[3].z;
            verts[3].x = tmp.x; verts[3].y = tmp.y; verts[3].z = tmp.z;
            if(surf.numverts&LAYER_DUP) loopk(4) 
            {
                vertinfo &v = verts[k], &b = verts[k+4];
                b.x = v.x;
                b.y = v.y;
                b.z = v.z;
            }
        }
        surf.numverts = (surf.numverts & ~MAXFACEVERTS) | 3;
        if(vis&2)
        {
            verts[1] = verts[2]; verts[2] = verts[3];
            if(surf.numverts&LAYER_DUP) { verts[3] = verts[4]; verts[4] = verts[6]; verts[5] = verts[7]; }
        }
        else if(surf.numverts&LAYER_DUP) { verts[3] = verts[4]; verts[4] = verts[5]; verts[5] = verts[6]; }
    }
}

void fixrotatedlightmaps()
{
    loopi(8) fixrotatedlightmaps(worldroot[i], ivec(i, ivec(0, 0, 0), worldsize>>1), worldsize>>1);
}
