// worldio.cpp: loading & saving of maps and savegames

#include "engine.h"
#include "game.h"
#include "texture.h"
#include "state.h"

#define MAXTRANS 5000                  // max amount of data to swallow in 1 go

static void fixent(entity &e, int version)
{
    if(version <= 10 && e.type >= 7) e.type++;
    if(version <= 12 && e.type >= 8) e.type++;
    if(version <= 14 && e.type >= ET_MAPMODEL && e.type <= 16)
    {
        if(e.type == 16) e.type = ET_MAPMODEL;
        else e.type++;
    }
    if(version <= 20 && e.type >= ET_ENVMAP) e.type++;
    if(version <= 21 && e.type >= ET_PARTICLES) e.type++;
    if(version <= 22 && e.type >= ET_SOUND) e.type++;
    if(version <= 23 && e.type >= ET_SPOTLIGHT) e.type++;
    if(version <= 30 && (e.type == ET_MAPMODEL || e.type == ET_PLAYERSTART)) e.attr1 = (int(e.attr1)+180)%360;
    if(version <= 31 && e.type == ET_MAPMODEL) { int yaw = (int(e.attr1)%360 + 360)%360 + 7; e.attr1 = yaw - yaw%15; }
}

#ifndef STANDALONE
string ogzname, bakname, cfgname, picname;

int savebak = 2;

enum { OCTSAV_CHILDREN = 0, OCTSAV_EMPTY, OCTSAV_SOLID, OCTSAV_NORMAL, OCTSAV_LODCUBE };

static int savemapprogress = 0;

void savec(cube *c, const ivec &o, int size, stream *f, bool nolms)
{
    loopi(8)
    {
        ivec co(i, o, size);
        if(c[i].children)
        {
            f->putchar(OCTSAV_CHILDREN);
            savec(c[i].children, co, size>>1, f, nolms);
        }
        else
        {
            int oflags = 0, surfmask = 0, totalverts = 0;
            if(c[i].material!=MAT_AIR) oflags |= 0x40;
            if(isempty(c[i])) f->putchar(oflags | OCTSAV_EMPTY);
            else
            {
                if(!nolms)
                {
                    if(c[i].merged) oflags |= 0x80;
                    if(c[i].ext) loopj(6)
                    {
                        const surfaceinfo &surf = c[i].ext->surfaces[j];
                        if(!surf.used()) continue;
                        oflags |= 0x20;
                        surfmask |= 1<<j;
                        totalverts += surf.totalverts();
                    }
                }

                if(isentirelysolid(c[i])) f->putchar(oflags | OCTSAV_SOLID);
                else
                {
                    f->putchar(oflags | OCTSAV_NORMAL);
                    f->write(c[i].edges, 12);
                }
            }

            loopj(6) f->putlil<ushort>(c[i].texture[j]);

            if(oflags&0x40) f->putlil<ushort>(c[i].material);
            if(oflags&0x80) f->putchar(c[i].merged);
            if(oflags&0x20)
            {
                f->putchar(surfmask);
                f->putchar(totalverts);
                loopj(6) if(surfmask&(1<<j))
                {
                    surfaceinfo surf = c[i].ext->surfaces[j];
                    vertinfo *verts = c[i].ext->verts() + surf.verts;
                    int layerverts = surf.numverts&MAXFACEVERTS, numverts = surf.totalverts(),
                        vertmask = 0, vertorder = 0, uvorder = 0,
                        dim = dimension(j), vc = C[dim], vr = R[dim];
                    if(numverts)
                    {
                        if(c[i].merged&(1<<j))
                        {
                            vertmask |= 0x04;
                            if(layerverts == 4)
                            {
                                ivec v[4] = { verts[0].getxyz(), verts[1].getxyz(), verts[2].getxyz(), verts[3].getxyz() };
                                loopk(4)
                                {
                                    const ivec &v0 = v[k], &v1 = v[(k+1)&3], &v2 = v[(k+2)&3], &v3 = v[(k+3)&3];
                                    if(v1[vc] == v0[vc] && v1[vr] == v2[vr] && v3[vc] == v2[vc] && v3[vr] == v0[vr])
                                    {
                                        vertmask |= 0x01;
                                        vertorder = k;
                                        break;
                                    }
                                }
                            }
                        }
                        else
                        {
                            int vis = visibletris(c[i], j, co, size);
                            if(vis&4 || faceconvexity(c[i], j) < 0) vertmask |= 0x01;
                            if(layerverts < 4 && vis&2) vertmask |= 0x02;
                        }
                        bool matchnorm = true;
                        loopk(numverts)
                        {
                            const vertinfo &v = verts[k];
                            if(v.u || v.v) vertmask |= 0x40;
                            if(v.norm) { vertmask |= 0x80; if(v.norm != verts[0].norm) matchnorm = false; }
                        }
                        if(matchnorm) vertmask |= 0x08;
                        if(vertmask&0x40 && layerverts == 4)
                        {
                            loopk(4)
                            {
                                const vertinfo &v0 = verts[k], &v1 = verts[(k+1)&3], &v2 = verts[(k+2)&3], &v3 = verts[(k+3)&3];
                                if(v1.u == v0.u && v1.v == v2.v && v3.u == v2.u && v3.v == v0.v)
                                {
                                    if(surf.numverts&LAYER_DUP)
                                    {
                                        const vertinfo &b0 = verts[4+k], &b1 = verts[4+((k+1)&3)], &b2 = verts[4+((k+2)&3)], &b3 = verts[4+((k+3)&3)];
                                        if(b1.u != b0.u || b1.v != b2.v || b3.u != b2.u || b3.v != b0.v)
                                            continue;
                                    }
                                    uvorder = k;
                                    vertmask |= 0x02 | (((k+4-vertorder)&3)<<4);
                                    break;
                                }
                            }
                        }
                    }
                    surf.verts = vertmask;
                    f->write(&surf, sizeof(surfaceinfo));
                    bool hasxyz = (vertmask&0x04)!=0, hasuv = (vertmask&0x40)!=0, hasnorm = (vertmask&0x80)!=0;
                    if(layerverts == 4)
                    {
                        if(hasxyz && vertmask&0x01)
                        {
                            ivec v0 = verts[vertorder].getxyz(), v2 = verts[(vertorder+2)&3].getxyz();
                            f->putlil<ushort>(v0[vc]); f->putlil<ushort>(v0[vr]);
                            f->putlil<ushort>(v2[vc]); f->putlil<ushort>(v2[vr]);
                            hasxyz = false;
                        }
                        if(hasuv && vertmask&0x02)
                        {
                            const vertinfo &v0 = verts[uvorder], &v2 = verts[(uvorder+2)&3];
                            f->putlil<ushort>(v0.u); f->putlil<ushort>(v0.v);
                            f->putlil<ushort>(v2.u); f->putlil<ushort>(v2.v);
                            if(surf.numverts&LAYER_DUP)
                            {
                                const vertinfo &b0 = verts[4+uvorder], &b2 = verts[4+((uvorder+2)&3)];
                                f->putlil<ushort>(b0.u); f->putlil<ushort>(b0.v);
                                f->putlil<ushort>(b2.u); f->putlil<ushort>(b2.v);
                            }
                            hasuv = false;
                        }
                    }
                    if(hasnorm && vertmask&0x08) { f->putlil<ushort>(verts[0].norm); hasnorm = false; }
                    if(hasxyz || hasuv || hasnorm) loopk(layerverts)
                    {
                        const vertinfo &v = verts[(k+vertorder)%layerverts];
                        if(hasxyz)
                        {
                            ivec xyz = v.getxyz();
                            f->putlil<ushort>(xyz[vc]); f->putlil<ushort>(xyz[vr]);
                        }
                        if(hasuv) { f->putlil<ushort>(v.u); f->putlil<ushort>(v.v); }
                        if(hasnorm) f->putlil<ushort>(v.norm);
                    }
                    if(surf.numverts&LAYER_DUP) loopk(layerverts)
                    {
                        const vertinfo &v = verts[layerverts + (k+vertorder)%layerverts];
                        if(hasuv) { f->putlil<ushort>(v.u); f->putlil<ushort>(v.v); }
                    }
                }
            }
        }
    }
}

void loadvslot(stream *f, VSlot &vs, int changed)
{
    vs.changed = changed;
    if(vs.changed & (1<<VSLOT_SHPARAM))
    {
        int numparams = f->getlil<ushort>();
        string name;
        loopi(numparams)
        {
            SlotShaderParam &p = vs.params.add();
            int nlen = f->getlil<ushort>();
            f->read(name, min(nlen, MAXSTRLEN-1));
            name[min(nlen, MAXSTRLEN-1)] = '\0';
            if(nlen >= MAXSTRLEN) f->seek(nlen - (MAXSTRLEN-1), SEEK_CUR);
            p.name = getshaderparamname(name);
            p.loc = -1;
            loopk(4) p.val[k] = f->getlil<float>();
        }
    }
    if(vs.changed & (1<<VSLOT_SCALE)) vs.scale = f->getlil<float>();
    if(vs.changed & (1<<VSLOT_ROTATION)) vs.rotation = clamp(f->getlil<int>(), 0, 7);
    if(vs.changed & (1<<VSLOT_OFFSET))
    {
        vs.offset.x = f->getlil<int>();
        vs.offset.y = f->getlil<int>();
    }
    if(vs.changed & (1<<VSLOT_SCROLL))
    {
        vs.scroll.x = f->getlil<float>();
        vs.scroll.y = f->getlil<float>();
    }
    if(vs.changed & (1<<VSLOT_LAYER)) vs.layer = f->getlil<int>();
    if(vs.changed & (1<<VSLOT_ALPHA))
    {
        vs.alphafront = f->getlil<float>();
        vs.alphaback = f->getlil<float>();
    }
    if(vs.changed & (1<<VSLOT_COLOR))
    {
        loopk(3) vs.colorscale[k] = f->getlil<float>();
    }
}

void loadvslots(stream *f, int numvslots)
{
    int *prev = new int[numvslots];
    if(!prev) return;
    memset(prev, -1, numvslots*sizeof(int));
    while(numvslots > 0)
    {
        int changed = f->getlil<int>();
        if(changed < 0)
        {
            loopi(-changed) (*vslots).add(new VSlot(NULL, (*vslots).length()));
            numvslots += changed;
        }
        else
        {
            prev[(*vslots).length()] = f->getlil<int>();
            loadvslot(f, *(*vslots).add(new VSlot(NULL, (*vslots).length())), changed);
            numvslots--;
        }
    }
    loopv((*vslots)) if((*vslots).inrange(prev[i])) (*vslots)[prev[i]]->next = (*vslots)[i];
    delete[] prev;
}


void savevslot(stream *f, VSlot &vs, int prev)
{
    f->putlil<int>(vs.changed);
    f->putlil<int>(prev);
    if(vs.changed & (1<<VSLOT_SHPARAM))
    {
        f->putlil<ushort>(vs.params.length());
        loopv(vs.params)
        {
            SlotShaderParam &p = vs.params[i];
            f->putlil<ushort>(strlen(p.name));
            f->write(p.name, strlen(p.name));
            loopk(4) f->putlil<float>(p.val[k]);
        }
    }
    if(vs.changed & (1<<VSLOT_SCALE)) f->putlil<float>(vs.scale);
    if(vs.changed & (1<<VSLOT_ROTATION)) f->putlil<int>(vs.rotation);
    if(vs.changed & (1<<VSLOT_OFFSET))
    {
        f->putlil<int>(vs.offset.x);
        f->putlil<int>(vs.offset.y);
    }
    if(vs.changed & (1<<VSLOT_SCROLL))
    {
        f->putlil<float>(vs.scroll.x);
        f->putlil<float>(vs.scroll.y);
    }
    if(vs.changed & (1<<VSLOT_LAYER)) f->putlil<int>(vs.layer);
    if(vs.changed & (1<<VSLOT_ALPHA))
    {
        f->putlil<float>(vs.alphafront);
        f->putlil<float>(vs.alphaback);
    }
    if(vs.changed & (1<<VSLOT_COLOR))
    {
        loopk(3) f->putlil<float>(vs.colorscale[k]);
    }
}

void savevslots(stream *f, int numvslots)
{
    if(vslots->empty()) return;
    int *prev = new int[numvslots];
    memset(prev, -1, numvslots*sizeof(int));
    loopi(numvslots)
    {
        VSlot *vs = (*vslots)[i];
        if(vs->changed) continue;
        for(;;)
        {
            VSlot *cur = vs;
            do vs = vs->next; while(vs && vs->index >= numvslots);
            if(!vs) break;
            prev[vs->index] = cur->index;
        }
    }
    int lastroot = 0;
    loopi(numvslots)
    {
        VSlot &vs = *(*vslots)[i];
        if(!vs.changed) continue;
        if(lastroot < i) f->putlil<int>(-(i - lastroot));
        savevslot(f, vs, prev[i]);
        lastroot = i+1;
    }
    if(lastroot < numvslots) f->putlil<int>(-(numvslots - lastroot));
    delete[] prev;
}


struct surfacecompat
{
    uchar texcoords[8];
    uchar w, h;
    ushort x, y;
    uchar lmid, layer;
};

struct normalscompat
{
    bvec normals[4];
};

struct mergecompat
{
    ushort u1, u2, v1, v2;
};

cube *loadchildren(stream *f, const ivec &co, int size, bool &failed);

void convertoldsurfaces(cube &c, const ivec &co, int size, surfacecompat *srcsurfs, int hassurfs, normalscompat *normals, int hasnorms, mergecompat *merges, int hasmerges)
{
    surfaceinfo dstsurfs[6];
    vertinfo verts[6*2*MAXFACEVERTS];
    int totalverts = 0, numsurfs = 6;
    memset(dstsurfs, 0, sizeof(dstsurfs));
    loopi(6) if((hassurfs|hasnorms|hasmerges)&(1<<i))
    {
        surfaceinfo &dst = dstsurfs[i];
        vertinfo *curverts = NULL;
        int numverts = 0;
        surfacecompat *src = NULL, *blend = NULL;
        if(hassurfs&(1<<i))
        {
            src = &srcsurfs[i];
            if(src->layer&2)
            {
                blend = &srcsurfs[numsurfs++];
                dst.lmid[0] = src->lmid;
                dst.lmid[1] = blend->lmid;
                dst.numverts |= LAYER_BLEND;
                if(blend->lmid >= LMID_RESERVED && (src->x != blend->x || src->y != blend->y || src->w != blend->w || src->h != blend->h || memcmp(src->texcoords, blend->texcoords, sizeof(src->texcoords))))
                    dst.numverts |= LAYER_DUP;
            }
            else if(src->layer == 1) { dst.lmid[1] = src->lmid; dst.numverts |= LAYER_BOTTOM; }
            else { dst.lmid[0] = src->lmid; dst.numverts |= LAYER_TOP; }
        }
        else dst.numverts |= LAYER_TOP;
        bool uselms = hassurfs&(1<<i) && (dst.lmid[0] >= LMID_RESERVED || dst.lmid[1] >= LMID_RESERVED || dst.numverts&~LAYER_TOP),
             usemerges = hasmerges&(1<<i) && merges[i].u1 < merges[i].u2 && merges[i].v1 < merges[i].v2,
             usenorms = hasnorms&(1<<i) && normals[i].normals[0] != bvec(128, 128, 128);
        if(uselms || usemerges || usenorms)
        {
            ivec v[4], pos[4], e1, e2, e3, n, vo = ivec(co).mask(0xFFF).shl(3);
            genfaceverts(c, i, v);
            n.cross((e1 = v[1]).sub(v[0]), (e2 = v[2]).sub(v[0]));
            if(usemerges)
            {
                const mergecompat &m = merges[i];
                int offset = -n.dot(v[0].mul(size).add(vo)),
                    dim = dimension(i), vc = C[dim], vr = R[dim];
                loopk(4)
                {
                    const ivec &coords = facecoords[i][k];
                    int cc = coords[vc] ? m.u2 : m.u1,
                        rc = coords[vr] ? m.v2 : m.v1,
                        dc = n[dim] ? -(offset + n[vc]*cc + n[vr]*rc)/n[dim] : vo[dim];
                    ivec &mv = pos[k];
                    mv[vc] = cc;
                    mv[vr] = rc;
                    mv[dim] = dc;
                }
            }
            else
            {
                int convex = (e3 = v[0]).sub(v[3]).dot(n), vis = 3;
                if(!convex)
                {
                    if(ivec().cross(e3, e2).iszero()) { if(!n.iszero()) vis = 1; }
                    else if(n.iszero()) vis = 2;
                }
                int order = convex < 0 ? 1 : 0;
                pos[0] = v[order].mul(size).add(vo);
                pos[1] = vis&1 ? v[order+1].mul(size).add(vo) : pos[0];
                pos[2] = v[order+2].mul(size).add(vo);
                pos[3] = vis&2 ? v[(order+3)&3].mul(size).add(vo) : pos[0];
            }
            curverts = verts + totalverts;
            loopk(4)
            {
                if(k > 0 && (pos[k] == pos[0] || pos[k] == pos[k-1])) continue;
                vertinfo &dv = curverts[numverts++];
                dv.setxyz(pos[k]);
                if(uselms)
                {
                    float u = src->x + (src->texcoords[k*2] / 255.0f) * (src->w - 1),
                          v = src->y + (src->texcoords[k*2+1] / 255.0f) * (src->h - 1);
                    dv.u = ushort(floor(clamp((u) * float(USHRT_MAX+1)/LM_PACKW + 0.5f, 0.0f, float(USHRT_MAX))));
                    dv.v = ushort(floor(clamp((v) * float(USHRT_MAX+1)/LM_PACKH + 0.5f, 0.0f, float(USHRT_MAX))));
                }
                else dv.u = dv.v = 0;
                dv.norm = usenorms && normals[i].normals[k] != bvec(128, 128, 128) ? encodenormal(normals[i].normals[k].tonormal().normalize()) : 0;
            }
            dst.verts = totalverts;
            dst.numverts |= numverts;
            totalverts += numverts;
            if(dst.numverts&LAYER_DUP) loopk(4)
            {
                if(k > 0 && (pos[k] == pos[0] || pos[k] == pos[k-1])) continue;
                vertinfo &bv = verts[totalverts++];
                bv.setxyz(pos[k]);
                bv.u = ushort(floor(clamp((blend->x + (blend->texcoords[k*2] / 255.0f) * (blend->w - 1)) * float(USHRT_MAX+1)/LM_PACKW, 0.0f, float(USHRT_MAX))));
                bv.v = ushort(floor(clamp((blend->y + (blend->texcoords[k*2+1] / 255.0f) * (blend->h - 1)) * float(USHRT_MAX+1)/LM_PACKH, 0.0f, float(USHRT_MAX))));
                bv.norm = usenorms && normals[i].normals[k] != bvec(128, 128, 128) ? encodenormal(normals[i].normals[k].tonormal().normalize()) : 0;
            }
        }
    }
    setsurfaces(c, dstsurfs, verts, totalverts);
}

static inline int convertoldmaterial(int mat)
{
    return ((mat&7)<<MATF_VOLUME_SHIFT) | (((mat>>3)&3)<<MATF_CLIP_SHIFT) | (((mat>>5)&7)<<MATF_FLAG_SHIFT);
}

void loadc(stream *f, cube &c, const ivec &co, int size, bool &failed)
{
    bool haschildren = false;
    int octsav = f->getchar();
    switch(octsav&0x7)
    {
        case OCTSAV_CHILDREN:
            c.children = loadchildren(f, co, size>>1, failed);
            return;

        case OCTSAV_LODCUBE: haschildren = true;    break;
        case OCTSAV_EMPTY:  emptyfaces(c);          break;
        case OCTSAV_SOLID:  solidfaces(c);          break;
        case OCTSAV_NORMAL: f->read(c.edges, 12); break;
        default: failed = true; return;
    }
    loopi(6) c.texture[i] = mapversion<14 ? f->getchar() : f->getlil<ushort>();
    if(mapversion < 7) f->seek(3, SEEK_CUR);
    else if(mapversion <= 31)
    {
        uchar mask = f->getchar();
        if(mask & 0x80)
        {
            int mat = f->getchar();
            if(mapversion < 27)
            {
                static const ushort matconv[] = { MAT_AIR, MAT_WATER, MAT_CLIP, MAT_GLASS|MAT_CLIP, MAT_NOCLIP, MAT_LAVA|MAT_DEATH, MAT_GAMECLIP, MAT_DEATH };
                c.material = size_t(mat) < sizeof(matconv)/sizeof(matconv[0]) ? matconv[mat] : MAT_AIR;
            }
            else c.material = convertoldmaterial(mat);
        }
        surfacecompat surfaces[12];
        normalscompat normals[6];
        mergecompat merges[6];
        int hassurfs = 0, hasnorms = 0, hasmerges = 0;
        if(mask & 0x3F)
        {
            int numsurfs = 6;
            loopi(numsurfs)
            {
                if(i >= 6 || mask & (1 << i))
                {
                    f->read(&surfaces[i], sizeof(surfacecompat));
                    lilswap(&surfaces[i].x, 2);
                    if(mapversion < 10) ++surfaces[i].lmid;
                    if(mapversion < 18)
                    {
                        if(surfaces[i].lmid >= LMID_AMBIENT1) ++surfaces[i].lmid;
                        if(surfaces[i].lmid >= LMID_BRIGHT1) ++surfaces[i].lmid;
                    }
                    if(mapversion < 19)
                    {
                        if(surfaces[i].lmid >= LMID_DARK) surfaces[i].lmid += 2;
                    }
                    if(i < 6)
                    {
                        if(mask & 0x40) { hasnorms |= 1<<i; f->read(&normals[i], sizeof(normalscompat)); }
                        if(surfaces[i].layer != 0 || surfaces[i].lmid != LMID_AMBIENT)
                            hassurfs |= 1<<i;
                        if(surfaces[i].layer&2) numsurfs++;
                    }
                }
            }
        }
        if(mapversion <= 8) edgespan2vectorcube(c);
        if(mapversion <= 11)
        {
            swap(c.faces[0], c.faces[2]);
            swap(c.texture[0], c.texture[4]);
            swap(c.texture[1], c.texture[5]);
            if(hassurfs&0x33)
            {
                swap(surfaces[0], surfaces[4]);
                swap(surfaces[1], surfaces[5]);
                hassurfs = (hassurfs&~0x33) | ((hassurfs&0x30)>>4) | ((hassurfs&0x03)<<4);
            }
        }
        if(mapversion >= 20)
        {
            if(octsav&0x80)
            {
                int merged = f->getchar();
                c.merged = merged&0x3F;
                if(merged&0x80)
                {
                    int mask = f->getchar();
                    if(mask)
                    {
                        hasmerges = mask&0x3F;
                        loopi(6) if(mask&(1<<i))
                        {
                            mergecompat *m = &merges[i];
                            f->read(m, sizeof(mergecompat));
                            lilswap(&m->u1, 4);
                            if(mapversion <= 25)
                            {
                                int uorigin = m->u1 & 0xE000, vorigin = m->v1 & 0xE000;
                                m->u1 = (m->u1 - uorigin) << 2;
                                m->u2 = (m->u2 - uorigin) << 2;
                                m->v1 = (m->v1 - vorigin) << 2;
                                m->v2 = (m->v2 - vorigin) << 2;
                            }
                        }
                    }
                }
            }
        }
        if(hassurfs || hasnorms || hasmerges)
            convertoldsurfaces(c, co, size, surfaces, hassurfs, normals, hasnorms, merges, hasmerges);
    }
    else
    {
        if(octsav&0x40)
        {
            if(mapversion <= 32)
            {
                int mat = f->getchar();
                c.material = convertoldmaterial(mat);
            }
            else c.material = f->getlil<ushort>();
        }
        if(octsav&0x80) c.merged = f->getchar();
        if(octsav&0x20)
        {
            int surfmask, totalverts;
            surfmask = f->getchar();
            totalverts = max(f->getchar(), 0);
            newcubeext(c, totalverts, false);
            memset(c.ext->surfaces, 0, sizeof(c.ext->surfaces));
            memset(c.ext->verts(), 0, totalverts*sizeof(vertinfo));
            int offset = 0;
            loopi(6) if(surfmask&(1<<i))
            {
                surfaceinfo &surf = c.ext->surfaces[i];
                f->read(&surf, sizeof(surfaceinfo));
                int vertmask = surf.verts, numverts = surf.totalverts();
                if(!numverts) { surf.verts = 0; continue; }
                surf.verts = offset;
                vertinfo *verts = c.ext->verts() + offset;
                offset += numverts;
                ivec v[4], n, vo = ivec(co).mask(0xFFF).shl(3);
                int layerverts = surf.numverts&MAXFACEVERTS, dim = dimension(i), vc = C[dim], vr = R[dim], bias = 0;
                genfaceverts(c, i, v);
                bool hasxyz = (vertmask&0x04)!=0, hasuv = (vertmask&0x40)!=0, hasnorm = (vertmask&0x80)!=0;
                if(hasxyz)
                {
                    ivec e1, e2, e3;
                    n.cross((e1 = v[1]).sub(v[0]), (e2 = v[2]).sub(v[0]));
                    if(n.iszero()) n.cross(e2, (e3 = v[3]).sub(v[0]));
                    bias = -n.dot(ivec(v[0]).mul(size).add(vo));
                }
                else
                {
                    int vis = layerverts < 4 ? (vertmask&0x02 ? 2 : 1) : 3, order = vertmask&0x01 ? 1 : 0, k = 0;
                    verts[k++].setxyz(v[order].mul(size).add(vo));
                    if(vis&1) verts[k++].setxyz(v[order+1].mul(size).add(vo));
                    verts[k++].setxyz(v[order+2].mul(size).add(vo));
                    if(vis&2) verts[k++].setxyz(v[(order+3)&3].mul(size).add(vo));
                }
                if(layerverts == 4)
                {
                    if(hasxyz && vertmask&0x01)
                    {
                        ushort c1 = f->getlil<ushort>(), r1 = f->getlil<ushort>(), c2 = f->getlil<ushort>(), r2 = f->getlil<ushort>();
                        ivec xyz;
                        xyz[vc] = c1; xyz[vr] = r1; xyz[dim] = n[dim] ? -(bias + n[vc]*xyz[vc] + n[vr]*xyz[vr])/n[dim] : vo[dim];
                        verts[0].setxyz(xyz);
                        xyz[vc] = c1; xyz[vr] = r2; xyz[dim] = n[dim] ? -(bias + n[vc]*xyz[vc] + n[vr]*xyz[vr])/n[dim] : vo[dim];
                        verts[1].setxyz(xyz);
                        xyz[vc] = c2; xyz[vr] = r2; xyz[dim] = n[dim] ? -(bias + n[vc]*xyz[vc] + n[vr]*xyz[vr])/n[dim] : vo[dim];
                        verts[2].setxyz(xyz);
                        xyz[vc] = c2; xyz[vr] = r1; xyz[dim] = n[dim] ? -(bias + n[vc]*xyz[vc] + n[vr]*xyz[vr])/n[dim] : vo[dim];
                        verts[3].setxyz(xyz);
                        hasxyz = false;
                    }
                    if(hasuv && vertmask&0x02)
                    {
                        int uvorder = (vertmask&0x30)>>4;
                        vertinfo &v0 = verts[uvorder], &v1 = verts[(uvorder+1)&3], &v2 = verts[(uvorder+2)&3], &v3 = verts[(uvorder+3)&3];
                        v0.u = f->getlil<ushort>(); v0.v = f->getlil<ushort>();
                        v2.u = f->getlil<ushort>(); v2.v = f->getlil<ushort>();
                        v1.u = v0.u; v1.v = v2.v;
                        v3.u = v2.u; v3.v = v0.v;
                        if(surf.numverts&LAYER_DUP)
                        {
                            vertinfo &b0 = verts[4+uvorder], &b1 = verts[4+((uvorder+1)&3)], &b2 = verts[4+((uvorder+2)&3)], &b3 = verts[4+((uvorder+3)&3)];
                            b0.u = f->getlil<ushort>(); b0.v = f->getlil<ushort>();
                            b2.u = f->getlil<ushort>(); b2.v = f->getlil<ushort>();
                            b1.u = b0.u; b1.v = b2.v;
                            b3.u = b2.u; b3.v = b0.v;
                        }
                        hasuv = false;
                    }
                }
                if(hasnorm && vertmask&0x08)
                {
                    ushort norm = f->getlil<ushort>();
                    loopk(layerverts) verts[k].norm = norm;
                    hasnorm = false;
                }
                if(hasxyz || hasuv || hasnorm) loopk(layerverts)
                {
                    vertinfo &v = verts[k];
                    if(hasxyz)
                    {
                        ivec xyz;
                        xyz[vc] = f->getlil<ushort>(); xyz[vr] = f->getlil<ushort>();
                        xyz[dim] = n[dim] ? -(bias + n[vc]*xyz[vc] + n[vr]*xyz[vr])/n[dim] : vo[dim];
                        v.setxyz(xyz);
                    }
                    if(hasuv) { v.u = f->getlil<ushort>(); v.v = f->getlil<ushort>(); }
                    if(hasnorm) v.norm = f->getlil<ushort>();
                }
                if(surf.numverts&LAYER_DUP) loopk(layerverts)
                {
                    vertinfo &v = verts[k+layerverts], &t = verts[k];
                    v.setxyz(t.x, t.y, t.z);
                    if(hasuv) { v.u = f->getlil<ushort>(); v.v = f->getlil<ushort>(); }
                    v.norm = t.norm;
                }
            }
        }
    }

    c.children = (haschildren ? loadchildren(f, co, size>>1, failed) : NULL);
}

cube *loadchildren(stream *f, const ivec &co, int size, bool &failed)
{
    cube *c = newcubes();
    loopi(8)
    {
;       loadc(f, c[i], ivec(i, co, size), size, failed);
        if(failed) break;
    }
    return c;
}

struct bufstream : stream
{
    ucharbuf buf;

    bufstream(void * data, size_t len) : buf((uchar*)data, len) {}
    ~bufstream() {}

    void close() {
    }

    bool end() {
        return buf.remaining() > 0;
    }

    offset tell()
    {
        return buf.len;
    }

    bool getline(char *str, size_t len) { return false; }

    offset size()
    {
        return buf.maxlen;
    }

    bool seek(offset pos, int whence)
    {
        if (whence == SEEK_CUR) {
            buf.len = max(0, min((int)(buf.len + pos), (int)buf.maxlen));
            return true;
        }

        return false;
    }

    size_t read(void *out, size_t len) { return (int) buf.get((uchar*) out, len); }
    size_t write(const void *src, size_t len) { buf.put((uchar*) src, len); return len; }

    size_t printf(const char *fmt, ...)
    {
        return 0;
    }
};

void setworldsize(int size)
{
    worldsize = size;
    worldscale = 0;
    while(1<<worldscale < size) worldscale++;
}

size_t partial_save_world(
        void *p,
        size_t len,
        MapState *state,
        int _worldsize
)
{
    bufstream buf(p, len);
    bufstream *f = &buf;

    vslots = state->vslots;
    worldroot = state->root;
    slots = state->slots;

    setworldsize(_worldsize);

    // TODO
    bool nolms = true;

    int numvslots = state->vslots->length();
    if(!nolms)
    {
        numvslots = compactvslots();
        allchanged();
    }

    savevslots(f, numvslots);

    savec(worldroot, ivec(0, 0, 0), worldsize>>1, f, nolms);

    if(!nolms)
    {
        loopv(lightmaps)
        {
            LightMap &lm = lightmaps[i];
            f->putchar(lm.type | (lm.unlitx>=0 ? 0x80 : 0));
            if(lm.unlitx>=0)
            {
                f->putlil<ushort>(ushort(lm.unlitx));
                f->putlil<ushort>(ushort(lm.unlity));
            }
            f->write(lm.data, lm.bpp*LM_PACKW*LM_PACKH);
        }
    }

    return buf.buf.len;
}

MapState *empty_world(int scale)
{
    MapState *state = new MapState;
    state->vslots = new vector<VSlot*>;
    state->slots = new vector<Slot*>;
    vslots = state->vslots;
    slots = state->slots;
    worldroot = newcubes(F_EMPTY);
    emptymap(scale, true, NULL);
    state->root = worldroot;

    return state;
}

MapState *partial_load_world(
        void *p,
        size_t len,
        int numvslots,
        int _worldsize,
        int _mapversion,
        int numlightmaps,
        int numpvs,
        int blendmap
)
{
    bufstream buf(p, len);
    bufstream *f = &buf;

    mapversion = _mapversion;
    setworldsize(_worldsize);

    MapState *state = new MapState;
    state->vslots = new vector<VSlot*>;
    state->slots = new vector<Slot*>;

    vslots = state->vslots;
    slots = state->slots;

    loadvslots(f, numvslots);

    bool failed = false;
    worldroot = loadchildren(f, ivec(0, 0, 0), worldsize>>1, failed);
    if(failed) return NULL;

    state->root = worldroot;

    validatec(worldroot, worldsize>>1);

    if(mapversion >= 7) loopi(numlightmaps)
    {
        LightMap &lm = lightmaps.add();
        if(mapversion >= 17)
        {
            int type = f->getchar();
            lm.type = type&0x7F;
            if(mapversion >= 20 && type&0x80)
            {
                lm.unlitx = f->getlil<ushort>();
                lm.unlity = f->getlil<ushort>();
            }
        }
        if(lm.type&LM_ALPHA && (lm.type&LM_TYPE)!=LM_BUMPMAP1) lm.bpp = 4;
        lm.data = new uchar[lm.bpp*LM_PACKW*LM_PACKH];
        f->read(lm.data, lm.bpp * LM_PACKW * LM_PACKH);
        lm.finalize();
    }

    if(mapversion >= 25 && numpvs > 0) loadpvs(f, numpvs);
    if(mapversion >= 28 && blendmap) loadblendmap(f, blendmap);

    //identflags |= IDF_OVERRIDDEN;
    //execfile("data/default_map_settings.cfg", false);
    //execfile(cfgname, false);
    //identflags &= ~IDF_OVERRIDDEN;

    extern void fixlightmapnormals();
    if(mapversion <= 25) fixlightmapnormals();
    extern void fixrotatedlightmaps();
    if(mapversion <= 31) fixrotatedlightmaps();

    return state;
}

cube *loadchildren_buf(void *p, size_t len, int size, int _mapversion)
{
    bool failed = false;
    bufstream buf(p, len);
    mapversion = _mapversion;

    cube *c = loadchildren(&buf, ivec(0, 0, 0), size>>1, failed);
    if (failed) {
        return NULL;
    }

    return c;
}

size_t savec_buf(void *p, unsigned int len, cube *c, int size)
{
    bufstream buf(p, len);
    savec(c, ivec(0, 0, 0), size>>1, &buf, false);
    return buf.buf.len;
}

cube *getcubeindex(cube *c, int i)
{
    return &c[i];
}

void cube_setedge(cube *c, int i, uchar value)
{
    c->edges[i] = value;
}

void cube_settexture(cube *c, int i, ushort value)
{
    c->texture[i] = value;
}

int getnumvslots(MapState *state)
{
    return state->vslots->length();
}

VSlot *getvslotindex(MapState *state, int i)
{
    return (*state->vslots)[i];
}

void filtertext(char *dst, const char *src, bool whitespace, bool forcespace, size_t len)
{
    for(int c = uchar(*src); c; c = uchar(*++src))
    {
        if(c == '\f')
        {
            if(!*++src) break;
            continue;
        }
        if(!iscubeprint(c))
        {
            if(!iscubespace(c) || !whitespace) continue;
            if(forcespace) c = ' ';
        }
        *dst++ = c;
        if(!--len) break;
    }
    *dst = '\0';
}

int processedits(ucharbuf &p)
{
    static char text[MAXTRANS];
    int type;
    editinfo *edit = NULL;

    while(p.remaining()) {
        int type = getint(p);
        switch(type)
    {
        case N_CLIPBOARD:
        {
            int cn = getint(p), unpacklen = getint(p), packlen = getint(p);
            ucharbuf q = p.subbuf(max(packlen, 0));
            unpackeditinfo(edit, q.buf, q.maxlen, unpacklen);
            break;
        }
        case N_UNDO:
        case N_REDO:
        {
            int cn = getint(p), unpacklen = getint(p), packlen = getint(p);
            ucharbuf q = p.subbuf(max(packlen, 0));
            unpackundo(q.buf, q.maxlen, unpacklen);
            break;
        }

        case N_NEWMAP:
        {
            int size = getint(p);
            if(size>=0) emptymap(size, true, NULL);
            else enlargemap(true);
            break;
        }

        case N_EDITF:              // coop editing messages
        case N_EDITT:
        case N_EDITM:
        case N_FLIP:
        case N_COPY:
        case N_PASTE:
        case N_ROTATE:
        case N_REPLACE:
        case N_DELCUBE:
        case N_EDITVSLOT:
        {
            selinfo sel;
            sel.o.x = getint(p); sel.o.y = getint(p); sel.o.z = getint(p);
            sel.s.x = getint(p); sel.s.y = getint(p); sel.s.z = getint(p);
            sel.grid = getint(p); sel.orient = getint(p);
            sel.cx = getint(p); sel.cxs = getint(p); sel.cy = getint(p), sel.cys = getint(p);
            sel.corner = getint(p);
            switch(type)
            {
                case N_EDITF: { int dir = getint(p), mode = getint(p); if(sel.validate()) mpeditface(dir, mode, sel, false); break; }
                case N_EDITT:
                {
                    int tex = getint(p),
                        allfaces = getint(p);
                    if(p.remaining() < 2) return -1;
                    int extra = lilswap(*(const ushort *)p.pad(2));
                    if(p.remaining() < extra) return -1;
                    ucharbuf ebuf = p.subbuf(extra);
                    if(sel.validate()) mpedittex(tex, allfaces, sel, ebuf);
                    break;
                }
                case N_EDITM: { int mat = getint(p), filter = getint(p); if(sel.validate()) mpeditmat(mat, filter, sel, false); break; }
                case N_FLIP: if(sel.validate()) mpflip(sel, false); break;
                case N_COPY: {
                    if(sel.validate()) {
                        mpcopy(edit, sel, false);
                    }
                    break;
                }
                case N_PASTE: if(sel.validate()) mppaste(edit, sel, false); break;
                case N_ROTATE: { int dir = getint(p); if(sel.validate()) mprotate(dir, sel, false); break; }
                case N_REPLACE:
                {
                    int oldtex = getint(p),
                        newtex = getint(p),
                        insel = getint(p);
                    if(p.remaining() < 2) return -1;
                    int extra = lilswap(*(const ushort *)p.pad(2));
                    if(p.remaining() < extra) return -1;
                    ucharbuf ebuf = p.subbuf(extra);
                    if(sel.validate()) mpreplacetex(oldtex, newtex, insel>0, sel, ebuf);
                    break;
                }
                case N_DELCUBE: if(sel.validate()) mpdelcube(sel, false); break;
                case N_EDITVSLOT:
                {
                    int delta = getint(p),
                        allfaces = getint(p);
                    if(p.remaining() < 2) return -1;
                    int extra = lilswap(*(const ushort *)p.pad(2));
                    if(p.remaining() < extra) return -1;
                    ucharbuf ebuf = p.subbuf(extra);
                    if(sel.validate()) mpeditvslot(delta, allfaces, sel, ebuf);
                    break;
                }
            }
            break;
        }

        case N_REMIP:
            {
                mpremip(false);
                break;
            }
        case N_EDITENT:            // coop edit of ent
            {
                int i = getint(p);
                float x = getint(p)/DMF, y = getint(p)/DMF, z = getint(p)/DMF;
                int type = getint(p);
                int attr1 = getint(p), attr2 = getint(p), attr3 = getint(p), attr4 = getint(p), attr5 = getint(p);

                // HANDLED IN GO
                //mpeditent(i, vec(x, y, z), type, attr1, attr2, attr3, attr4, attr5, false);
                break;
            }

        case N_EDITVAR:
        {
            int type = getint(p);
            getstring(text, p);
            string name;
            filtertext(name, text, false);

            // HANDLED IN GO
            switch(type)
            {
                case ID_VAR:
                {
                    getint(p);
                    break;
                }
                case ID_FVAR:
                {
                    getfloat(p);
                    break;
                }
                case ID_SVAR:
                {
                    getstring(text, p);
                    break;
                }
            }
            break;
        }

        default:
            printf("got unknown message type=%d", type);
            return -1;
    }
    }

    return 0;
}

void setup_state(MapState *state)
{
    worldroot = state->root;
    vslots = state->vslots;
    slots = state->slots;
}

void teardown_state(MapState *state)
{
    state->root = worldroot;
    state->vslots = vslots;
    state->slots = slots;
    worldroot = NULL;
    vslots = NULL;
    slots = NULL;
}

bool apply_messages(MapState *state, int _worldsize, void *data, size_t len)
{
    setup_state(state);

    setworldsize(_worldsize);

    ucharbuf buf((uchar*)data, len);
    int result = processedits(buf);
    if (result == -1) {
        return false;
    }

    teardown_state(state);
    return true;
}

editinfo *store_copy(MapState *state, void *data, size_t len)
{
    ucharbuf p((uchar*)data, len);
    getint(p); // type
    selinfo sel;
    sel.o.x = getint(p); sel.o.y = getint(p); sel.o.z = getint(p);
    sel.s.x = getint(p); sel.s.y = getint(p); sel.s.z = getint(p);
    sel.grid = getint(p); sel.orient = getint(p);
    sel.cx = getint(p); sel.cxs = getint(p); sel.cy = getint(p), sel.cys = getint(p);
    sel.corner = getint(p);
    if(!sel.validate()) return NULL;
    editinfo *edit = NULL;
    setup_state(state);
    mpcopy(edit, sel, false);
    teardown_state(state);
    return edit;
}

bool apply_paste(MapState *state, editinfo *info, void *data, size_t len)
{
    ucharbuf p((uchar*)data, len);
    getint(p); // type
    selinfo sel;
    sel.o.x = getint(p); sel.o.y = getint(p); sel.o.z = getint(p);
    sel.s.x = getint(p); sel.s.y = getint(p); sel.s.z = getint(p);
    sel.grid = getint(p); sel.orient = getint(p);
    sel.cx = getint(p); sel.cxs = getint(p); sel.cy = getint(p), sel.cys = getint(p);
    sel.corner = getint(p);
    if(!sel.validate()) return false;
    setup_state(state);
    mppaste(info, sel, false);
    teardown_state(state);
    return true;
}

void free_state(MapState *state)
{
    freeocta(state->root);
    state->slots->setsize(0);
    state->vslots->setsize(0);
}

void free_edit(editinfo *info)
{
    freeeditinfo(info);
}

bool load_texture_index(void *data, size_t len, MapState *state)
{
    ucharbuf p((uchar*)data, len);

    vslots = state->vslots;
    slots = state->slots;

    int numchanges = getint(p);
    char textype[MAXTRANS];
    char name[MAXTRANS];
    for (int i = 0; i < numchanges; i++) {
        int type = getint(p);

        if (p.overread()) {
            return false;
        }

        switch (type) {
            case 0: {
                getstring(textype, p);
                getstring(name, p);
                int rotation = getint(p),
                    xoffset = getint(p),
                    yoffset = getint(p);
                float scale = getfloat(p);
                texture(textype, name, &rotation, &xoffset, &yoffset, &scale);
                break;
            }
            case 1: {
                int limit = getint(p);
                texturereset(&limit);
                break;
            }
        }
    }

    return true;
}

int dbgvars = 0;

#endif

