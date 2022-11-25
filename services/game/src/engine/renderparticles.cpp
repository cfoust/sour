// renderparticles.cpp

#include "engine.h"
#include "rendertarget.h"

Shader *particleshader = NULL, *particlenotextureshader = NULL;

VARP(particlesize, 20, 100, 500);
    
// Check canemitparticles() to limit the rate that paricles can be emitted for models/sparklies
// Automatically stops particles being emitted when paused or in reflective drawing
VARP(emitmillis, 1, 17, 1000);
static int lastemitframe = 0, emitoffset = 0;
static bool canemit = false, regenemitters = false, canstep = false;

static bool canemitparticles()
{
    if(reflecting || refracting) return false;
    return canemit || emitoffset;
}

VARP(showparticles, 0, 1, 1);
VAR(cullparticles, 0, 1, 1);
VAR(replayparticles, 0, 1, 1);
VARN(seedparticles, seedmillis, 0, 3000, 10000);
VAR(dbgpcull, 0, 0, 1);
VAR(dbgpseed, 0, 0, 1);

struct particleemitter
{
    extentity *ent;
    vec bbmin, bbmax;
    vec center;
    float radius;
    ivec cullmin, cullmax;
    int maxfade, lastemit, lastcull;

    particleemitter(extentity *ent)
        : ent(ent), bbmin(ent->o), bbmax(ent->o), maxfade(-1), lastemit(0), lastcull(0)
    {}

    void finalize()
    {
        center = vec(bbmin).add(bbmax).mul(0.5f);
        radius = bbmin.dist(bbmax)/2;
        cullmin = ivec(int(floor(bbmin.x)), int(floor(bbmin.y)), int(floor(bbmin.z)));
        cullmax = ivec(int(ceil(bbmax.x)), int(ceil(bbmax.y)), int(ceil(bbmax.z)));
        if(dbgpseed) conoutf(CON_DEBUG, "radius: %f, maxfade: %d", radius, maxfade);
    }
    
    void extendbb(const vec &o, float size = 0)
    {
        bbmin.x = min(bbmin.x, o.x - size);
        bbmin.y = min(bbmin.y, o.y - size);
        bbmin.z = min(bbmin.z, o.z - size);
        bbmax.x = max(bbmax.x, o.x + size);
        bbmax.y = max(bbmax.y, o.y + size);
        bbmax.z = max(bbmax.z, o.z + size);
    }

    void extendbb(float z, float size = 0)
    {
        bbmin.z = min(bbmin.z, z - size);
        bbmax.z = max(bbmax.z, z + size);
    }
};

static vector<particleemitter> emitters;
static particleemitter *seedemitter = NULL;

void clearparticleemitters()
{
    emitters.setsize(0);
    regenemitters = true;
}

void addparticleemitters()
{
    emitters.setsize(0);
    const vector<extentity *> &ents = entities::getents();
    loopv(ents)
    {
        extentity &e = *ents[i];
        if(e.type != ET_PARTICLES) continue;
        emitters.add(particleemitter(&e));
    }
    regenemitters = false;
}

enum
{
    PT_PART = 0,
    PT_TAPE,
    PT_TRAIL,
    PT_TEXT,
    PT_TEXTICON,
    PT_METER,
    PT_METERVS,
    PT_FIREBALL,
    PT_LIGHTNING,
    PT_FLARE,

    PT_MOD    = 1<<8,
    PT_RND4   = 1<<9,
    PT_LERP   = 1<<10, // use very sparingly - order of blending issues
    PT_TRACK  = 1<<11,
    PT_GLARE  = 1<<12,
    PT_SOFT   = 1<<13,
    PT_HFLIP  = 1<<14,
    PT_VFLIP  = 1<<15,
    PT_ROT    = 1<<16,
    PT_CULL   = 1<<17,
    PT_FEW    = 1<<18,
    PT_ICON   = 1<<19,
    PT_NOTEX  = 1<<20,
    PT_SHADER = 1<<21,
    PT_FLIP  = PT_HFLIP | PT_VFLIP | PT_ROT
};

const char *partnames[] = { "part", "tape", "trail", "text", "texticon", "meter", "metervs", "fireball", "lightning", "flare" };

struct particle
{
    vec o, d;
    int gravity, fade, millis;
    bvec color;
    uchar flags;
    float size;
    union
    {
        const char *text;
        float val;
        physent *owner;
        struct
        {
            uchar color2[3];
            uchar progress;
        };
    }; 
};

struct partvert
{
    vec pos;
    bvec4 color;
    vec2 tc;
};

#define COLLIDERADIUS 8.0f
#define COLLIDEERROR 1.0f

struct partrenderer
{
    Texture *tex;
    const char *texname;
    int texclamp;
    uint type;
    int collide;
    string info;
   
    partrenderer(const char *texname, int texclamp, int type, int collide = 0)
        : tex(NULL), texname(texname), texclamp(texclamp), type(type), collide(collide)
    {
    }
    partrenderer(int type, int collide = 0)
        : tex(NULL), texname(NULL), texclamp(0), type(type), collide(collide)
    {
    }
    virtual ~partrenderer()
    {
    }

    virtual void init(int n) { }
    virtual void reset() = 0;
    virtual void resettracked(physent *owner) { }   
    virtual particle *addpart(const vec &o, const vec &d, int fade, int color, float size, int gravity = 0) = 0;    
    virtual int adddepthfx(vec &bbmin, vec &bbmax) { return 0; }
    virtual void update() { }
    virtual void render() = 0;
    virtual bool haswork() = 0;
    virtual int count() = 0; //for debug
    virtual void cleanup() {}

    virtual void seedemitter(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int gravity)
    {
    }

    //blend = 0 => remove it
    void calc(particle *p, int &blend, int &ts, vec &o, vec &d, bool step = true)
    {
        o = p->o;
        d = p->d;
        if(type&PT_TRACK && p->owner) game::particletrack(p->owner, o, d);
        if(p->fade <= 5) 
        {
            ts = 1;
            blend = 255;
        }
        else
        {
            ts = lastmillis-p->millis;
            blend = max(255 - (ts<<8)/p->fade, 0);
            if(p->gravity)
            {
                if(ts > p->fade) ts = p->fade;
                float t = ts;
                o.add(vec(d).mul(t/5000.0f));
                o.z -= t*t/(2.0f * 5000.0f * p->gravity);
            }
            if(collide && o.z < p->val && step)
            {
                if(collide >= 0)
                {
                    vec surface;
                    float floorz = rayfloor(vec(o.x, o.y, p->val), surface, RAY_CLIPMAT, COLLIDERADIUS);
                    float collidez = floorz<0 ? o.z-COLLIDERADIUS : p->val - floorz;
                    if(o.z >= collidez+COLLIDEERROR) 
                        p->val = collidez+COLLIDEERROR;
                    else 
                    {
                        adddecal(collide, vec(o.x, o.y, collidez), vec(p->o).sub(o).normalize(), 2*p->size, p->color, type&PT_RND4 ? (p->flags>>5)&3 : 0);
                        blend = 0;
                    }
                }
                else blend = 0;
            }
        }
    }

    const char *debuginfo()
    {
        formatstring(info, "%d\t%s(", count(), partnames[type&0xFF]);
        if(type&PT_GLARE) concatstring(info, "g,");
        if(type&PT_SOFT) concatstring(info, "s,");
        if(type&PT_LERP) concatstring(info, "l,");
        if(type&PT_MOD) concatstring(info, "m,");
        if(type&PT_RND4) concatstring(info, "r,");
        if(type&PT_FLIP) concatstring(info, "f,");
        if(collide) concatstring(info, "c,");
        int len = strlen(info);
        info[len-1] = info[len-1] == ',' ? ')' : '\0';
        if(texname)
        {   
            const char *title = strrchr(texname, '/');
            if(title) concformatstring(info, ": %s", title+1);
        }
        return info;
    }
};

struct listparticle : particle
{   
    listparticle *next;
};

VARP(outlinemeters, 0, 0, 1);

struct listrenderer : partrenderer
{
    static listparticle *parempty;
    listparticle *list;

    listrenderer(const char *texname, int texclamp, int type, int collide = 0) 
        : partrenderer(texname, texclamp, type, collide), list(NULL)
    {
    }
    listrenderer(int type, int collide = 0)
        : partrenderer(type, collide), list(NULL)
    {
    }

    virtual ~listrenderer()
    {
    }

    virtual void killpart(listparticle *p)
    {
    }

    void reset()  
    {
        if(!list) return;
        listparticle *p = list;
        for(;;)
        {
            killpart(p);
            if(p->next) p = p->next;
            else break;
        }
        p->next = parempty;
        parempty = list;
        list = NULL;
    }
    
    void resettracked(physent *owner) 
    {
        if(!(type&PT_TRACK)) return;
        for(listparticle **prev = &list, *cur = list; cur; cur = *prev)
        {
            if(!owner || cur->owner==owner) 
            {
                *prev = cur->next;
                cur->next = parempty;
                parempty = cur;
            }
            else prev = &cur->next;
        }
    }
    
    particle *addpart(const vec &o, const vec &d, int fade, int color, float size, int gravity) 
    {
        if(!parempty)
        {
            listparticle *ps = new listparticle[256];
            loopi(255) ps[i].next = &ps[i+1];
            ps[255].next = parempty;
            parempty = ps;
        }
        listparticle *p = parempty;
        parempty = p->next;
        p->next = list;
        list = p;
        p->o = o;
        p->d = d;
        p->gravity = gravity;
        p->fade = fade;
        p->millis = lastmillis + emitoffset;
        p->color = bvec(color>>16, (color>>8)&0xFF, color&0xFF);
        p->size = size;
        p->owner = NULL;
        p->flags = 0;
        return p;
    }
    
    int count() 
    {
        int num = 0;
        listparticle *lp;
        for(lp = list; lp; lp = lp->next) num++;
        return num;
    }
    
    bool haswork() 
    {
        return (list != NULL);
    }
    
    virtual void startrender() = 0;
    virtual void endrender() = 0;
    virtual void renderpart(listparticle *p, const vec &o, const vec &d, int blend, int ts) = 0;

    void render() 
    {
        startrender();
        if(texname)
        {
            if(!tex) tex = textureload(texname, texclamp);
            glBindTexture(GL_TEXTURE_2D, tex->id);
        }
        
        for(listparticle **prev = &list, *p = list; p; p = *prev)
        {   
            vec o, d;
            int blend, ts;
            calc(p, blend, ts, o, d, canstep);
            if(blend > 0) 
            {
                renderpart(p, o, d, blend, ts);

                if(p->fade > 5 || !canstep) 
                {
                    prev = &p->next;
                    continue;
                }
            }
            //remove
            *prev = p->next;
            p->next = parempty;
            killpart(p);
            parempty = p;
        }
       
        endrender();
    }
};

listparticle *listrenderer::parempty = NULL;

struct meterrenderer : listrenderer
{
    meterrenderer(int type)
        : listrenderer(type|PT_NOTEX|PT_LERP)
    {}

    void startrender()
    {
        glDisable(GL_BLEND);
        gle::defvertex();
    }

    void endrender()
    {
        glEnable(GL_BLEND);
    }

    void renderpart(listparticle *p, const vec &o, const vec &d, int blend, int ts)
    {
        int basetype = type&0xFF;
        float scale = FONTH*p->size/80.0f, right = 8, left = p->progress/100.0f*right;
        matrix4x3 m(camright, vec(camup).neg(), vec(camdir).neg(), o);
        m.scale(scale);
        m.translate(-right/2.0f, 0, 0);

        if(outlinemeters)
        {
            gle::colorf(0, 0.8f, 0);
            gle::begin(GL_TRIANGLE_STRIP);
            loopk(10)
            {
                const vec2 &sc = sincos360[k*(180/(10-1))];
                float c = (0.5f + 0.1f)*sc.y, s = 0.5f - (0.5f + 0.1f)*sc.x;
                gle::attrib(m.transform(vec2(-c, s)));
                gle::attrib(m.transform(vec2(right + c, s)));
            }
            gle::end();
        }

        if(basetype==PT_METERVS) gle::colorub(p->color2[0], p->color2[1], p->color2[2]);
        else gle::colorf(0, 0, 0);
        gle::begin(GL_TRIANGLE_STRIP);
        loopk(10)
        {
            const vec2 &sc = sincos360[k*(180/(10-1))];
            float c = 0.5f*sc.y, s = 0.5f - 0.5f*sc.x;
            gle::attrib(m.transform(vec2(left + c, s)));
            gle::attrib(m.transform(vec2(right + c, s)));
        }
        gle::end();

        if(outlinemeters)
        {
            gle::colorf(0, 0.8f, 0);
            gle::begin(GL_TRIANGLE_FAN);
            loopk(10)
            {
                const vec2 &sc = sincos360[k*(180/(10-1))];
                float c = (0.5f + 0.1f)*sc.y, s = 0.5f - (0.5f + 0.1f)*sc.x;
                gle::attrib(m.transform(vec2(left + c, s)));
            }
            gle::end();
        }

        gle::color(p->color);
        gle::begin(GL_TRIANGLE_STRIP);
        loopk(10)
        {
            const vec2 &sc = sincos360[k*(180/(10-1))];
            float c = 0.5f*sc.y, s = 0.5f - 0.5f*sc.x;
            gle::attrib(m.transform(vec2(-c, s)));
            gle::attrib(m.transform(vec2(left + c, s)));
        }
        gle::end();
    }
};
static meterrenderer meters(PT_METER), metervs(PT_METERVS);

struct textrenderer : listrenderer
{
    textrenderer(int type)
        : listrenderer(type)
    {}

    void startrender()
    {
    }

    void endrender()
    {
    }

    void killpart(listparticle *p)
    {
        if(p->text && p->flags&1) delete[] p->text;
    }

    void renderpart(listparticle *p, const vec &o, const vec &d, int blend, int ts)
    {
        float scale = p->size/80.0f, xoff = -(text_width(p->text) + ((p->flags>>1)&7)*FONTH)/2, yoff = 0;

        matrix4x3 m(camright, vec(camup).neg(), vec(camdir).neg(), o);
        m.scale(scale);
        m.translate(xoff, yoff, 50);

        textmatrix = &m;
        draw_text(p->text, 0, 0, p->color.r, p->color.g, p->color.b, blend);
        textmatrix = NULL;
    } 
};
static textrenderer texts(PT_TEXT|PT_LERP);

struct texticonrenderer : listrenderer
{
    texticonrenderer(const char *texname, int type)
        : listrenderer(texname, 3, type)
    {}

    void startrender()
    {
        gle::defvertex();
        gle::deftexcoord0();
        gle::defcolor(4, GL_UNSIGNED_BYTE);
        gle::begin(GL_QUADS);
    }

    void endrender()
    {
        gle::end();
    }

    void renderpart(listparticle *p, const vec &o, const vec &d, int blend, int ts)
    {
        float scale = p->size/80.0f, xoff = p->val, yoff = 0;

        matrix4x3 m(camright, vec(camup).neg(), vec(camdir).neg(), o);
        m.scale(scale);
        m.translate(xoff, yoff, 50);

        float tx = 0.25f*(p->flags&3), ty = 0.25f*((p->flags>>2)&3);

        gle::attrib(m.transform(vec2(0, 0)));
            gle::attrib(tx, ty);
            gle::attrib(p->color, blend);
        gle::attrib(m.transform(vec2(FONTH, 0)));
            gle::attrib(tx + 0.25f, ty);
            gle::attrib(p->color, blend);
        gle::attrib(m.transform(vec2(FONTH, FONTH)));
            gle::attrib(tx + 0.25f, ty + 0.25f);
            gle::attrib(p->color, blend);
        gle::attrib(m.transform(vec2(0, FONTH)));
            gle::attrib(tx, ty + 0.25f);
            gle::attrib(p->color, blend);
    }
};
static texticonrenderer texticons("packages/hud/items.png", PT_TEXTICON|PT_LERP);

template<int T>
static inline void modifyblend(const vec &o, int &blend)
{
    blend = min(blend<<2, 255);
}

template<>
inline void modifyblend<PT_TAPE>(const vec &o, int &blend)
{
}

template<int T>
static inline void genpos(const vec &o, const vec &d, float size, int grav, int ts, partvert *vs)
{
    vec udir = vec(camup).sub(camright).mul(size);
    vec vdir = vec(camup).add(camright).mul(size);
    vs[0].pos = vec(o.x + udir.x, o.y + udir.y, o.z + udir.z);
    vs[1].pos = vec(o.x + vdir.x, o.y + vdir.y, o.z + vdir.z);
    vs[2].pos = vec(o.x - udir.x, o.y - udir.y, o.z - udir.z);
    vs[3].pos = vec(o.x - vdir.x, o.y - vdir.y, o.z - vdir.z);
}

template<>
inline void genpos<PT_TAPE>(const vec &o, const vec &d, float size, int ts, int grav, partvert *vs)
{
    vec dir1 = d, dir2 = d, c;
    dir1.sub(o);
    dir2.sub(camera1->o);
    c.cross(dir2, dir1).normalize().mul(size);
    vs[0].pos = vec(d.x-c.x, d.y-c.y, d.z-c.z);
    vs[1].pos = vec(o.x-c.x, o.y-c.y, o.z-c.z);
    vs[2].pos = vec(o.x+c.x, o.y+c.y, o.z+c.z);
    vs[3].pos = vec(d.x+c.x, d.y+c.y, d.z+c.z);
}

template<>
inline void genpos<PT_TRAIL>(const vec &o, const vec &d, float size, int ts, int grav, partvert *vs)
{
    vec e = d;
    if(grav) e.z -= float(ts)/grav;
    e.div(-75.0f).add(o);
    genpos<PT_TAPE>(o, e, size, ts, grav, vs);
}

template<int T>
static inline void genrotpos(const vec &o, const vec &d, float size, int grav, int ts, partvert *vs, int rot)
{
    genpos<T>(o, d, size, grav, ts, vs);
}

#define ROTCOEFFS(n) { \
    vec(-1,  1, 0).rotate_around_z(n*2*M_PI/32.0f), \
    vec( 1,  1, 0).rotate_around_z(n*2*M_PI/32.0f), \
    vec( 1, -1, 0).rotate_around_z(n*2*M_PI/32.0f), \
    vec(-1, -1, 0).rotate_around_z(n*2*M_PI/32.0f) \
}
static const vec rotcoeffs[32][4] =
{
    ROTCOEFFS(0),  ROTCOEFFS(1),  ROTCOEFFS(2),  ROTCOEFFS(3),  ROTCOEFFS(4),  ROTCOEFFS(5),  ROTCOEFFS(6),  ROTCOEFFS(7),
    ROTCOEFFS(8),  ROTCOEFFS(9),  ROTCOEFFS(10), ROTCOEFFS(11), ROTCOEFFS(12), ROTCOEFFS(13), ROTCOEFFS(14), ROTCOEFFS(15),
    ROTCOEFFS(16), ROTCOEFFS(17), ROTCOEFFS(18), ROTCOEFFS(19), ROTCOEFFS(20), ROTCOEFFS(21), ROTCOEFFS(22), ROTCOEFFS(7),
    ROTCOEFFS(24), ROTCOEFFS(25), ROTCOEFFS(26), ROTCOEFFS(27), ROTCOEFFS(28), ROTCOEFFS(29), ROTCOEFFS(30), ROTCOEFFS(31),
};

template<>
inline void genrotpos<PT_PART>(const vec &o, const vec &d, float size, int grav, int ts, partvert *vs, int rot)
{
    const vec *coeffs = rotcoeffs[rot];
    (vs[0].pos = o).add(vec(camright).mul(coeffs[0].x*size)).add(vec(camup).mul(coeffs[0].y*size));
    (vs[1].pos = o).add(vec(camright).mul(coeffs[1].x*size)).add(vec(camup).mul(coeffs[1].y*size));
    (vs[2].pos = o).add(vec(camright).mul(coeffs[2].x*size)).add(vec(camup).mul(coeffs[2].y*size));
    (vs[3].pos = o).add(vec(camright).mul(coeffs[3].x*size)).add(vec(camup).mul(coeffs[3].y*size));
}

template<int T>
static inline void seedpos(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int grav)
{
    if(grav)
    {
        vec end(o);
        float t = fade;
        end.add(vec(d).mul(t/5000.0f));
        end.z -= t*t/(2.0f * 5000.0f * grav);
        pe.extendbb(end, size);

        float tpeak = d.z*grav;
        if(tpeak > 0 && tpeak < fade) pe.extendbb(o.z + 1.5f*d.z*tpeak/5000.0f, size);
    }
}

template<>
inline void seedpos<PT_TAPE>(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int grav)
{
    pe.extendbb(d, size);
}

template<>
inline void seedpos<PT_TRAIL>(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int grav)
{
    vec e = d;
    if(grav) e.z -= float(fade)/grav;
    e.div(-75.0f).add(o);
    pe.extendbb(e, size); 
}

template<int T>
struct varenderer : partrenderer
{
    partvert *verts;
    particle *parts;
    int maxparts, numparts, lastupdate, rndmask;
    GLuint vbo;

    varenderer(const char *texname, int type, int collide = 0) 
        : partrenderer(texname, 3, type, collide),
          verts(NULL), parts(NULL), maxparts(0), numparts(0), lastupdate(-1), rndmask(0), vbo(0)
    {
        if(type & PT_HFLIP) rndmask |= 0x01;
        if(type & PT_VFLIP) rndmask |= 0x02;
        if(type & PT_ROT) rndmask |= 0x1F<<2;
        if(type & PT_RND4) rndmask |= 0x03<<5;
    }

    void cleanup()
    {
        if(vbo) { glDeleteBuffers_(1, &vbo); vbo = 0; }
    }
    
    void init(int n)
    {
        DELETEA(parts);
        DELETEA(verts);
        parts = new particle[n];
        verts = new partvert[n*4];
        maxparts = n;
        numparts = 0;
        lastupdate = -1;
    }
        
    void reset() 
    {
        numparts = 0;
        lastupdate = -1;
    }
    
    void resettracked(physent *owner) 
    {
        if(!(type&PT_TRACK)) return;
        loopi(numparts)
        {
            particle *p = parts+i;
            if(!owner || (p->owner == owner)) p->fade = -1;
        }
        lastupdate = -1;
    }
    
    int count() 
    {
        return numparts;
    }
    
    bool haswork() 
    {
        return (numparts > 0);
    }

    particle *addpart(const vec &o, const vec &d, int fade, int color, float size, int gravity) 
    {
        particle *p = parts + (numparts < maxparts ? numparts++ : rnd(maxparts)); //next free slot, or kill a random kitten
        p->o = o;
        p->d = d;
        p->gravity = gravity;
        p->fade = fade;
        p->millis = lastmillis + emitoffset;
        p->color = bvec(color>>16, (color>>8)&0xFF, color&0xFF);
        p->size = size;
        p->owner = NULL;
        p->flags = 0x80 | (rndmask ? rnd(0x80) & rndmask : 0);
        lastupdate = -1;
        return p;
    }
 
    void seedemitter(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int gravity)
    {
        pe.maxfade = max(pe.maxfade, fade);
        size *= SQRT2;
        pe.extendbb(o, size);

        seedpos<T>(pe, o, d, fade, size, gravity);
        if(!gravity) return;

        vec end(o);
        float t = fade;
        end.add(vec(d).mul(t/5000.0f));
        end.z -= t*t/(2.0f * 5000.0f * gravity);
        pe.extendbb(end, size);

        float tpeak = d.z*gravity;
        if(tpeak > 0 && tpeak < fade) pe.extendbb(o.z + 1.5f*d.z*tpeak/5000.0f, size);
    }
 
    void genverts(particle *p, partvert *vs, bool regen)
    {
        vec o, d;
        int blend, ts;

        calc(p, blend, ts, o, d);
        if(blend <= 1 || p->fade <= 5) p->fade = -1; //mark to remove on next pass (i.e. after render)

        modifyblend<T>(o, blend);

        if(regen)
        {
            p->flags &= ~0x80;

            #define SETTEXCOORDS(u1c, u2c, v1c, v2c, body) \
            { \
                float u1 = u1c, u2 = u2c, v1 = v1c, v2 = v2c; \
                body; \
                vs[0].tc = vec2(u1, v1); \
                vs[1].tc = vec2(u2, v1); \
                vs[2].tc = vec2(u2, v2); \
                vs[3].tc = vec2(u1, v2); \
            }
            if(type&PT_RND4)
            {
                float tx = 0.5f*((p->flags>>5)&1), ty = 0.5f*((p->flags>>6)&1);
                SETTEXCOORDS(tx, tx + 0.5f, ty, ty + 0.5f,
                {
                    if(p->flags&0x01) swap(u1, u2);
                    if(p->flags&0x02) swap(v1, v2);
                });
            } 
            else if(type&PT_ICON)
            {
                float tx = 0.25f*(p->flags&3), ty = 0.25f*((p->flags>>2)&3);
                SETTEXCOORDS(tx, tx + 0.25f, ty, ty + 0.25f, {});
            }
            else SETTEXCOORDS(0, 1, 0, 1, {});

            #define SETCOLOR(r, g, b, a) \
            do { \
                bvec4 col(r, g, b, a); \
                loopi(4) vs[i].color = col; \
            } while(0)
            #define SETMODCOLOR SETCOLOR((p->color.r*blend)>>8, (p->color.g*blend)>>8, (p->color.b*blend)>>8, 255)
            if(type&PT_MOD) SETMODCOLOR;
            else SETCOLOR(p->color.r, p->color.g, p->color.b, blend);
        }
        else if(type&PT_MOD) SETMODCOLOR;
        else loopi(4) vs[i].color.a = blend;

        if(type&PT_ROT) genrotpos<T>(o, d, p->size, ts, p->gravity, vs, (p->flags>>2)&0x1F);
        else genpos<T>(o, d, p->size, ts, p->gravity, vs);
    }

    void genverts()
    {
        loopi(numparts)
        {
            particle *p = &parts[i];
            partvert *vs = &verts[i*4];
            if(p->fade < 0)
            {
                do 
                {
                    --numparts; 
                    if(numparts <= i) return;
                }
                while(parts[numparts].fade < 0);
                *p = parts[numparts];
                genverts(p, vs, true);
            }
            else genverts(p, vs, (p->flags&0x80)!=0);
        }
    }
   
    void update()
    {
        if(lastmillis == lastupdate && vbo) return;
        lastupdate = lastmillis;

        genverts();

        if(!vbo) glGenBuffers_(1, &vbo);
        gle::bindvbo(vbo);
        glBufferData_(GL_ARRAY_BUFFER, maxparts*4*sizeof(partvert), NULL, GL_STREAM_DRAW);
        glBufferSubData_(GL_ARRAY_BUFFER, 0, numparts*4*sizeof(partvert), verts);
        gle::clearvbo();
    }
 
    void render()
    {   
        if(!tex) tex = textureload(texname, texclamp);
        glBindTexture(GL_TEXTURE_2D, tex->id);

        gle::bindvbo(vbo);
        const partvert *ptr = 0;
        gle::vertexpointer(sizeof(partvert), ptr->pos.v);
        gle::texcoord0pointer(sizeof(partvert), ptr->tc.v);
        gle::colorpointer(sizeof(partvert), ptr->color.v);
        gle::enablevertex();
        gle::enabletexcoord0();
        gle::enablecolor();
        gle::enablequads();

        gle::drawquads(0, numparts);

        gle::disablequads();
        gle::disablevertex();
        gle::disabletexcoord0();
        gle::disablecolor();
        gle::clearvbo();
    }
};
typedef varenderer<PT_PART> quadrenderer;
typedef varenderer<PT_TAPE> taperenderer;
typedef varenderer<PT_TRAIL> trailrenderer;

#include "depthfx.h"
#include "explosion.h"
#include "lensflare.h"
#include "lightning.h"

struct softquadrenderer : quadrenderer
{
    softquadrenderer(const char *texname, int type, int collide = 0)
        : quadrenderer(texname, type|PT_SOFT, collide)
    {
    }

    int adddepthfx(vec &bbmin, vec &bbmax)
    {
        if(!depthfxtex.highprecision() && !depthfxtex.emulatehighprecision()) return 0;
        int numsoft = 0;
        loopi(numparts)
        {
            particle &p = parts[i];
            float radius = p.size*SQRT2;
            vec o, d;
            int blend, ts;
            calc(&p, blend, ts, o, d, false);
            if(!isfoggedsphere(radius, p.o) && (depthfxscissor!=2 || depthfxtex.addscissorbox(p.o, radius))) 
            {
                numsoft++;
                loopk(3)
                {
                    bbmin[k] = min(bbmin[k], o[k] - radius);
                    bbmax[k] = max(bbmax[k], o[k] + radius);
                }
            }
        }
        return numsoft;
    }
};

static partrenderer *parts[] = 
{
    new quadrenderer("<grey>packages/particles/blood.png", PT_PART|PT_FLIP|PT_MOD|PT_RND4, DECAL_BLOOD), // blood spats (note: rgb is inverted) 
    new trailrenderer("packages/particles/base.png", PT_TRAIL|PT_LERP),                            // water, entity
    new quadrenderer("<grey>packages/particles/smoke.png", PT_PART|PT_FLIP|PT_LERP),               // smoke
    new quadrenderer("<grey>packages/particles/steam.png", PT_PART|PT_FLIP),                       // steam
    new quadrenderer("<grey>packages/particles/flames.png", PT_PART|PT_HFLIP|PT_RND4|PT_GLARE),    // flame on - no flipping please, they have orientation
    new quadrenderer("packages/particles/ball1.png", PT_PART|PT_FEW|PT_GLARE),                     // fireball1
    new quadrenderer("packages/particles/ball2.png", PT_PART|PT_FEW|PT_GLARE),                     // fireball2
    new quadrenderer("packages/particles/ball3.png", PT_PART|PT_FEW|PT_GLARE),                     // fireball3
    new taperenderer("packages/particles/flare.jpg", PT_TAPE|PT_GLARE),                            // streak
    &lightnings,                                                                                   // lightning
    &fireballs,                                                                                    // explosion fireball
    &bluefireballs,                                                                                // bluish explosion fireball
    new quadrenderer("packages/particles/spark.png", PT_PART|PT_FLIP|PT_GLARE),                    // sparks
    new quadrenderer("packages/particles/base.png",  PT_PART|PT_FLIP|PT_GLARE),                    // edit mode entities
    new quadrenderer("<grey>packages/particles/snow.png", PT_PART|PT_FLIP|PT_RND4, -1),            // colliding snow
    new quadrenderer("packages/particles/muzzleflash1.jpg", PT_PART|PT_FEW|PT_FLIP|PT_GLARE|PT_TRACK), // muzzle flash
    new quadrenderer("packages/particles/muzzleflash2.jpg", PT_PART|PT_FEW|PT_FLIP|PT_GLARE|PT_TRACK), // muzzle flash
    new quadrenderer("packages/particles/muzzleflash3.jpg", PT_PART|PT_FEW|PT_FLIP|PT_GLARE|PT_TRACK), // muzzle flash
    new quadrenderer("packages/hud/items.png", PT_PART|PT_FEW|PT_ICON),                            // hud icon
    new quadrenderer("<colorify:1/1/1>packages/hud/items.png", PT_PART|PT_FEW|PT_ICON),            // grey hud icon
    &texts,                                                                                        // text
    &texticons,                                                                                    // text icons
    &meters,                                                                                       // meter
    &metervs,                                                                                      // meter vs.
    &flares                                                                                        // lens flares - must be done last
};

void finddepthfxranges()
{
    depthfxmin = vec(1e16f, 1e16f, 1e16f);
    depthfxmax = vec(0, 0, 0);
    numdepthfxranges = fireballs.finddepthfxranges(depthfxowners, depthfxranges, 0, MAXDFXRANGES, depthfxmin, depthfxmax);
    numdepthfxranges = bluefireballs.finddepthfxranges(depthfxowners, depthfxranges, numdepthfxranges, MAXDFXRANGES, depthfxmin, depthfxmax);
    loopk(3)
    {
        depthfxmin[k] -= depthfxmargin;
        depthfxmax[k] += depthfxmargin;
    }
    if(depthfxparts)
    {
        loopi(sizeof(parts)/sizeof(parts[0]))
        {
            partrenderer *p = parts[i];
            if(p->type&PT_SOFT && p->adddepthfx(depthfxmin, depthfxmax))
            {
                if(!numdepthfxranges)
                {
                    numdepthfxranges = 1;
                    depthfxowners[0] = NULL;
                    depthfxranges[0] = 0;
                }
            }
        }
    }              
    if(depthfxscissor<2 && numdepthfxranges>0) depthfxtex.addscissorbox(depthfxmin, depthfxmax);
}
 
VARFP(maxparticles, 10, 4000, 40000, initparticles());
VARFP(fewparticles, 10, 100, 40000, initparticles());

void initparticles() 
{
    if(!particleshader) particleshader = lookupshaderbyname("particle");
    if(!particlenotextureshader) particlenotextureshader = lookupshaderbyname("particlenotexture");
    loopi(sizeof(parts)/sizeof(parts[0])) parts[i]->init(parts[i]->type&PT_FEW ? min(fewparticles, maxparticles) : maxparticles);
}

void clearparticles()
{   
    loopi(sizeof(parts)/sizeof(parts[0])) parts[i]->reset();
    clearparticleemitters();
}   

void cleanupparticles()
{
    loopi(sizeof(parts)/sizeof(parts[0])) parts[i]->cleanup();
}

void removetrackedparticles(physent *owner)
{
    loopi(sizeof(parts)/sizeof(parts[0])) parts[i]->resettracked(owner);
}

VARP(particleglare, 0, 2, 100);

VARN(debugparticles, dbgparts, 0, 0, 1);

void debugparticles()
{
    if(!dbgparts) return;
    int n = sizeof(parts)/sizeof(parts[0]);
    pushhudmatrix();
    hudmatrix.ortho(0, FONTH*n*2*screenw/float(screenh), FONTH*n*2, 0, -1, 1); //squeeze into top-left corner        
    flushhudmatrix();
    hudshader->set();
    loopi(n) draw_text(parts[i]->info, FONTH, (i+n/2)*FONTH);
    pophudmatrix();
}

void renderparticles(bool mainpass)
{
    canstep = mainpass;
    //want to debug BEFORE the lastpass render (that would delete particles)
    if(dbgparts && mainpass) loopi(sizeof(parts)/sizeof(parts[0])) parts[i]->debuginfo();

    if(glaring && !particleglare) return;
    
    loopi(sizeof(parts)/sizeof(parts[0])) 
    {
        if(glaring && !(parts[i]->type&PT_GLARE)) continue;
        parts[i]->update();
    }
    
    bool rendered = false;
    uint lastflags = PT_LERP|PT_SHADER,
         flagmask = PT_LERP|PT_MOD|PT_SHADER|PT_NOTEX;
   
    if(binddepthfxtex()) flagmask |= PT_SOFT;

    loopi(sizeof(parts)/sizeof(parts[0]))
    {
        partrenderer *p = parts[i];
        if(glaring && !(p->type&PT_GLARE)) continue;
        if(!p->haswork()) continue;
    
        if(!rendered)
        {
            rendered = true;
            glDepthMask(GL_FALSE);
            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);             

            if(glaring) GLOBALPARAMF(colorscale, particleglare, particleglare, particleglare, 1);
            else GLOBALPARAMF(colorscale, 1, 1, 1, 1);
        }
        
        uint flags = p->type & flagmask, changedbits = (flags ^ lastflags);
        if(changedbits)
        {
            if(changedbits&PT_LERP)
            {
                if(flags&PT_LERP) resetfogcolor();
                else zerofogcolor();
            }
            if(changedbits&(PT_LERP|PT_MOD))
            {
                if(flags&PT_LERP) glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
                else if(flags&PT_MOD) glBlendFunc(GL_ZERO, GL_ONE_MINUS_SRC_COLOR);
                else glBlendFunc(GL_SRC_ALPHA, GL_ONE);
            }
            if(!(flags&PT_SHADER))
            {
                if(changedbits&(PT_SOFT|PT_SHADER|PT_NOTEX|PT_LERP))
                {
                    if(flags&PT_SOFT)
                    {
                        if(!depthfxtex.highprecision()) SETSHADER(particlesoft8);
                        else SETSHADER(particlesoft);

                        binddepthfxparams(depthfxpartblend);
                    }
                    else if(flags&PT_NOTEX) particlenotextureshader->set();
                    else particleshader->set();
                }
            }
            lastflags = flags;        
        }
        p->render();
    }

    if(rendered)
    {
        if(lastflags&(PT_LERP|PT_MOD)) glBlendFunc(GL_SRC_ALPHA, GL_ONE);
        if(!(lastflags&PT_LERP)) resetfogcolor();
        glDisable(GL_BLEND);
        glDepthMask(GL_TRUE);
    }
}

static int addedparticles = 0;

static inline particle *newparticle(const vec &o, const vec &d, int fade, int type, int color, float size, int gravity = 0)
{
    static particle dummy;
    if(seedemitter) 
    {
        parts[type]->seedemitter(*seedemitter, o, d, fade, size, gravity);
        return &dummy;
    }
    if(fade + emitoffset < 0) return &dummy;
    addedparticles++;
    return parts[type]->addpart(o, d, fade, color, size, gravity);
}

VARP(maxparticledistance, 256, 1024, 4096);

static void splash(int type, int color, int radius, int num, int fade, const vec &p, float size, int gravity)
{
    if(camera1->o.dist(p) > maxparticledistance && !seedemitter) return;
    float collidez = parts[type]->collide ? p.z - raycube(p, vec(0, 0, -1), COLLIDERADIUS, RAY_CLIPMAT) + (parts[type]->collide >= 0 ? COLLIDEERROR : 0) : -1; 
    int fmin = 1;
    int fmax = fade*3;
    loopi(num)
    {
        int x, y, z;
        do
        {
            x = rnd(radius*2)-radius;
            y = rnd(radius*2)-radius;
            z = rnd(radius*2)-radius;
        }
        while(x*x+y*y+z*z>radius*radius);
    	vec tmp = vec((float)x, (float)y, (float)z);
        int f = (num < 10) ? (fmin + rnd(fmax)) : (fmax - (i*(fmax-fmin))/(num-1)); //help deallocater by using fade distribution rather than random
        newparticle(p, tmp, f, type, color, size, gravity)->val = collidez;
    }
}

extern "C" {

void bb_splash(int type, int color, int radius, int num, int fade, float p0, float p1, float p2, float size, int gravity)
{
  static vec temp;
  temp.x = p0; temp.y = p1; temp.z = p2;
  splash(type, color, radius, num, fade, temp, size, gravity);
}

}

static void regularsplash(int type, int color, int radius, int num, int fade, const vec &p, float size, int gravity, int delay = 0) 
{
    if(!canemitparticles() || (delay > 0 && rnd(delay) != 0)) return;
    splash(type, color, radius, num, fade, p, size, gravity);
}

bool canaddparticles()
{
    return !renderedgame && !shadowmapping && !minimized;
}

void regular_particle_splash(int type, int num, int fade, const vec &p, int color, float size, int radius, int gravity, int delay) 
{
    if(!canaddparticles()) return;
    regularsplash(type, color, radius, num, fade, p, size, gravity, delay);
}

void particle_splash(int type, int num, int fade, const vec &p, int color, float size, int radius, int gravity) 
{
    if(!canaddparticles()) return;
    splash(type, color, radius, num, fade, p, size, gravity);
}

VARP(maxtrail, 1, 500, 10000);

void particle_trail(int type, int fade, const vec &s, const vec &e, int color, float size, int gravity)
{
    if(!canaddparticles()) return;
    vec v;
    float d = e.dist(s, v);
    int steps = clamp(int(d*2), 1, maxtrail);
    v.div(steps);
    vec p = s;
    loopi(steps)
    {
        p.add(v);
        vec tmp = vec(float(rnd(11)-5), float(rnd(11)-5), float(rnd(11)-5));
        newparticle(p, tmp, rnd(fade)+fade, type, color, size, gravity);
    }
}

VARP(particletext, 0, 1, 1);
VARP(maxparticletextdistance, 0, 128, 10000);

void particle_text(const vec &s, const char *t, int type, int fade, int color, float size, int gravity, int icons)
{
    if(!canaddparticles()) return;
    if(!particletext || camera1->o.dist(s) > maxparticletextdistance) return;
    particle *p = newparticle(s, vec(0, 0, 1), fade, type, color, size, gravity);
    p->text = t;
    p->flags = icons<<1;
}

void particle_textcopy(const vec &s, const char *t, int type, int fade, int color, float size, int gravity)
{
    if(!canaddparticles()) return;
    if(!particletext || camera1->o.dist(s) > maxparticletextdistance) return;
    particle *p = newparticle(s, vec(0, 0, 1), fade, type, color, size, gravity);
    p->text = newstring(t);
    p->flags = 1;
}

void particle_texticon(const vec &s, int ix, int iy, float offset, int type, int fade, int color, float size, int gravity)
{
    if(!canaddparticles()) return;
    if(!particletext || camera1->o.dist(s) > maxparticletextdistance) return;
    particle *p = newparticle(s, vec(0, 0, 1), fade, type, color, size, gravity);
    p->flags |= ix | (iy<<2);
    p->val = offset;
}

void particle_icon(const vec &s, int ix, int iy, int type, int fade, int color, float size, int gravity)
{
    if(!canaddparticles()) return;
    particle *p = newparticle(s, vec(0, 0, 1), fade, type, color, size, gravity);
    p->flags |= ix | (iy<<2);
}

void particle_meter(const vec &s, float val, int type, int fade, int color, int color2, float size)
{
    if(!canaddparticles()) return;
    particle *p = newparticle(s, vec(0, 0, 1), fade, type, color, size);
    p->color2[0] = color2>>16;
    p->color2[1] = (color2>>8)&0xFF;
    p->color2[2] = color2&0xFF;
    p->progress = clamp(int(val*100), 0, 100);
}

void particle_flare(const vec &p, const vec &dest, int fade, int type, int color, float size, physent *owner)
{
    if(!canaddparticles()) return;
    newparticle(p, dest, fade, type, color, size)->owner = owner;
}

void particle_fireball(const vec &dest, float maxsize, int type, int fade, int color, float size)
{
    if(!canaddparticles()) return;
    float growth = maxsize - size;
    if(fade < 0) fade = int(growth*20);
    newparticle(dest, vec(0, 0, 1), fade, type, color, size)->val = growth;
}

//dir = 0..6 where 0=up
static inline vec offsetvec(vec o, int dir, int dist) 
{
    vec v = vec(o);    
    v[(2+dir)%3] += (dir>2)?(-dist):dist;
    return v;
}

//converts a 16bit color to 24bit
static inline int colorfromattr(int attr) 
{
    return (((attr&0xF)<<4) | ((attr&0xF0)<<8) | ((attr&0xF00)<<12)) + 0x0F0F0F;
}

/* Experiments in shapes...
 * dir: (where dir%3 is similar to offsetvec with 0=up)
 * 0..2 circle
 * 3.. 5 cylinder shell
 * 6..11 cone shell
 * 12..14 plane volume
 * 15..20 line volume, i.e. wall
 * 21 sphere
 * 24..26 flat plane
 * +32 to inverse direction
 */
void regularshape(int type, int radius, int color, int dir, int num, int fade, const vec &p, float size, int gravity, int vel = 200)
{
    if(!canemitparticles()) return;
    
    int basetype = parts[type]->type&0xFF;
    bool flare = (basetype == PT_TAPE) || (basetype == PT_LIGHTNING),
         inv = (dir&0x20)!=0, taper = (dir&0x40)!=0 && !seedemitter;
    dir &= 0x1F;
    loopi(num)
    {
        vec to, from;
        if(dir < 12) 
        { 
            const vec2 &sc = sincos360[rnd(360)];
            to[dir%3] = sc.y*radius;
            to[(dir+1)%3] = sc.x*radius;
            to[(dir+2)%3] = 0.0;
            to.add(p);
            if(dir < 3) //circle
                from = p;
            else if(dir < 6) //cylinder
            {
                from = to;
                to[(dir+2)%3] += radius;
                from[(dir+2)%3] -= radius;
            }
            else //cone
            {
                from = p;
                to[(dir+2)%3] += (dir < 9)?radius:(-radius);
            }
        }
        else if(dir < 15) //plane
        { 
            to[dir%3] = float(rnd(radius<<4)-(radius<<3))/8.0;
            to[(dir+1)%3] = float(rnd(radius<<4)-(radius<<3))/8.0;
            to[(dir+2)%3] = radius;
            to.add(p);
            from = to;
            from[(dir+2)%3] -= 2*radius;
        }
        else if(dir < 21) //line
        {
            if(dir < 18) 
            {
                to[dir%3] = float(rnd(radius<<4)-(radius<<3))/8.0;
                to[(dir+1)%3] = 0.0;
            } 
            else 
            {
                to[dir%3] = 0.0;
                to[(dir+1)%3] = float(rnd(radius<<4)-(radius<<3))/8.0;
            }
            to[(dir+2)%3] = 0.0;
            to.add(p);
            from = to;
            to[(dir+2)%3] += radius;  
        } 
        else if(dir < 24) //sphere
        {   
            to = vec(2*M_PI*float(rnd(1000))/1000.0, M_PI*float(rnd(1000)-500)/1000.0).mul(radius); 
            to.add(p);
            from = p;
        }
        else if(dir < 27) // flat plane
        {
            to[dir%3] = float(rndscale(2*radius)-radius);
            to[(dir+1)%3] = float(rndscale(2*radius)-radius);
            to[(dir+2)%3] = 0.0;
            to.add(p);
            from = to; 
        }
        else from = to = p; 

        if(inv) swap(from, to);

        if(taper)
        {
            float dist = clamp(from.dist2(camera1->o)/maxparticledistance, 0.0f, 1.0f);
            if(dist > 0.2f)
            {
                dist = 1 - (dist - 0.2f)/0.8f;
                if(rnd(0x10000) > dist*dist*0xFFFF) continue;
            }
        }
 
        if(flare)
            newparticle(from, to, rnd(fade*3)+1, type, color, size, gravity);
        else 
        {  
            vec d = vec(to).sub(from).rescale(vel); //velocity
            particle *n = newparticle(from, d, rnd(fade*3)+1, type, color, size, gravity);
            if(parts[type]->collide)
                n->val = from.z - raycube(from, vec(0, 0, -1), parts[type]->collide >= 0 ? COLLIDERADIUS : max(from.z, 0.0f), RAY_CLIPMAT) + (parts[type]->collide >= 0 ? COLLIDEERROR : 0);
        }
    }
}

static void regularflame(int type, const vec &p, float radius, float height, int color, int density = 3, float scale = 2.0f, float speed = 200.0f, float fade = 600.0f, int gravity = -15) 
{
    if(!canemitparticles()) return;
    
    float size = scale * min(radius, height);
    vec v(0, 0, min(1.0f, height)*speed);
    loopi(density)
    {
        vec s = p;        
        s.x += rndscale(radius*2.0f)-radius;
        s.y += rndscale(radius*2.0f)-radius;
        newparticle(s, v, rnd(max(int(fade*height), 1))+1, type, color, size, gravity);
    }
}

void regular_particle_flame(int type, const vec &p, float radius, float height, int color, int density, float scale, float speed, float fade, int gravity)
{
    if(!canaddparticles()) return;
    regularflame(type, p, radius, height, color, density, scale, speed, fade, gravity);
}

static void makeparticles(entity &e) 
{
    switch(e.attr1)
    {
        case 0: //fire and smoke -  <radius> <height> <rgb> - 0 values default to compat for old maps
        {
            //regularsplash(PART_FIREBALL1, 0xFFC8C8, 150, 1, 40, e.o, 4.8f);
            //regularsplash(PART_SMOKE, 0x897661, 50, 1, 200,  vec(e.o.x, e.o.y, e.o.z+3.0f), 2.4f, -20, 3);
            float radius = e.attr2 ? float(e.attr2)/100.0f : 1.5f,
                  height = e.attr3 ? float(e.attr3)/100.0f : radius/3;
            regularflame(PART_FLAME, e.o, radius, height, e.attr4 ? colorfromattr(e.attr4) : 0x903020, 3, 2.0f);
            regularflame(PART_SMOKE, vec(e.o.x, e.o.y, e.o.z + 4.0f*min(radius, height)), radius, height, 0x303020, 1, 4.0f, 100.0f, 2000.0f, -20);
            break;
        }
        case 1: //steam vent - <dir>
            regularsplash(PART_STEAM, 0x897661, 50, 1, 200, offsetvec(e.o, e.attr2, rnd(10)), 2.4f, -20);
            break;
        case 2: //water fountain - <dir>
        {
            int color;
            if(e.attr3 > 0) color = colorfromattr(e.attr3);
            else
            {
                int mat = MAT_WATER + clamp(-e.attr3, 0, 3); 
                const bvec &wfcol = getwaterfallcolor(mat);
                color = (int(wfcol[0])<<16) | (int(wfcol[1])<<8) | int(wfcol[2]);
                if(!color) 
                {
                    const bvec &wcol = getwatercolor(mat);
                    color = (int(wcol[0])<<16) | (int(wcol[1])<<8) | int(wcol[2]);
                }
            }
            regularsplash(PART_WATER, color, 150, 4, 200, offsetvec(e.o, e.attr2, rnd(10)), 0.6f, 2);
            break;
        }
        case 3: //fire ball - <size> <rgb>
            newparticle(e.o, vec(0, 0, 1), 1, PART_EXPLOSION, colorfromattr(e.attr3), 4.0f)->val = 1+e.attr2;
            break;
        case 4:  //tape - <dir> <length> <rgb>
        case 7:  //lightning 
        case 9:  //steam
        case 10: //water
        case 13: //snow
        {
            static const int typemap[]   = { PART_STREAK, -1, -1, PART_LIGHTNING, -1, PART_STEAM, PART_WATER, -1, -1, PART_SNOW };
            static const float sizemap[] = { 0.28f, 0.0f, 0.0f, 1.0f, 0.0f, 2.4f, 0.60f, 0.0f, 0.0f, 0.5f };
            static const int gravmap[] = { 0, 0, 0, 0, 0, -20, 2, 0, 0, 20 };
            int type = typemap[e.attr1-4];
            float size = sizemap[e.attr1-4];
            int gravity = gravmap[e.attr1-4];
            if(e.attr2 >= 256) regularshape(type, max(1+e.attr3, 1), colorfromattr(e.attr4), e.attr2-256, 5, e.attr5 > 0 ? min(int(e.attr5), 10000) : 200, e.o, size, gravity);
            else newparticle(e.o, offsetvec(e.o, e.attr2, max(1+e.attr3, 0)), 1, type, colorfromattr(e.attr4), size, gravity);
            break;
        }
        case 5: //meter, metervs - <percent> <rgb> <rgb2>
        case 6:
        {
            particle *p = newparticle(e.o, vec(0, 0, 1), 1, e.attr1==5 ? PART_METER : PART_METER_VS, colorfromattr(e.attr3), 2.0f);
            int color2 = colorfromattr(e.attr4);
            p->color2[0] = color2>>16;
            p->color2[1] = (color2>>8)&0xFF;
            p->color2[2] = color2&0xFF;
            p->progress = clamp(int(e.attr2), 0, 100);
            break;
        }
        case 11: // flame <radius> <height> <rgb> - radius=100, height=100 is the classic size
            regularflame(PART_FLAME, e.o, float(e.attr2)/100.0f, float(e.attr3)/100.0f, colorfromattr(e.attr4), 3, 2.0f);
            break;
        case 12: // smoke plume <radius> <height> <rgb>
            regularflame(PART_SMOKE, e.o, float(e.attr2)/100.0f, float(e.attr3)/100.0f, colorfromattr(e.attr4), 1, 4.0f, 100.0f, 2000.0f, -20);
            break;
        case 32: //lens flares - plain/sparkle/sun/sparklesun <red> <green> <blue>
        case 33:
        case 34:
        case 35:
            flares.addflare(e.o, e.attr2, e.attr3, e.attr4, (e.attr1&0x02)!=0, (e.attr1&0x01)!=0);
            break;
        default:
            if(!editmode)
            {
                defformatstring(ds, "particles %d?", e.attr1);
                particle_textcopy(e.o, ds, PART_TEXT, 1, 0x6496FF, 2.0f);
            }
            break;
    }
}

bool printparticles(extentity &e, char *buf, int len)
{
    switch(e.attr1)
    {
        case 0: case 4: case 7: case 8: case 9: case 10: case 11: case 12: case 13: 
            nformatstring(buf, len, "%s %d %d %d 0x%.3hX %d", entities::entname(e.type), e.attr1, e.attr2, e.attr3, e.attr4, e.attr5);
            return true;
        case 3:
            nformatstring(buf, len, "%s %d %d 0x%.3hX %d %d", entities::entname(e.type), e.attr1, e.attr2, e.attr3, e.attr4, e.attr5);
            return true;
        case 5: case 6:
            nformatstring(buf, len, "%s %d %d 0x%.3hX 0x%.3hX %d", entities::entname(e.type), e.attr1, e.attr2, e.attr3, e.attr4, e.attr5);
            return true; 
    }
    return false;
}

void seedparticles()
{
    renderprogress(0, "seeding particles");
    addparticleemitters();
    canemit = true;
    loopv(emitters)
    {
        particleemitter &pe = emitters[i];
        extentity &e = *pe.ent;
        seedemitter = &pe;
        for(int millis = 0; millis < seedmillis; millis += min(emitmillis, seedmillis/10))
            makeparticles(e);    
        seedemitter = NULL;
        pe.lastemit = -seedmillis;
        pe.finalize();
    }
}

void updateparticles()
{
    if(regenemitters) addparticleemitters();

    if(minimized) { canemit = false; return; }

    if(lastmillis - lastemitframe >= emitmillis)
    {
        canemit = true;
        lastemitframe = lastmillis - (lastmillis%emitmillis);
    }
    else canemit = false;
   
    flares.makelightflares();

    if(!editmode || showparticles) 
    {
        int emitted = 0, replayed = 0;
        addedparticles = 0;
        loopv(emitters)
        {
            particleemitter &pe = emitters[i];
            extentity &e = *pe.ent;
            if(e.o.dist(camera1->o) > maxparticledistance) { pe.lastemit = lastmillis; continue; } 
            if(cullparticles && pe.maxfade >= 0)
            {
                if(isfoggedsphere(pe.radius, pe.center)) { pe.lastcull = lastmillis; continue; }
                if(pvsoccluded(pe.cullmin, pe.cullmax)) { pe.lastcull = lastmillis; continue; }
            }
            makeparticles(e);
            emitted++;
            if(replayparticles && pe.maxfade > 5 && pe.lastcull > pe.lastemit)
            {
                for(emitoffset = max(pe.lastemit + emitmillis - lastmillis, -pe.maxfade); emitoffset < 0; emitoffset += emitmillis)
                {
                    makeparticles(e);
                    replayed++;
                }
                emitoffset = 0;
            } 
            pe.lastemit = lastmillis;
        }
        if(dbgpcull && (canemit || replayed) && addedparticles) conoutf(CON_DEBUG, "%d emitters, %d particles", emitted, addedparticles);
    }
    if(editmode) // show sparkly thingies for map entities in edit mode
    {
        const vector<extentity *> &ents = entities::getents();
        // note: order matters in this case as particles of the same type are drawn in the reverse order that they are added
        loopv(entgroup)
        {
            entity &e = *ents[entgroup[i]];
            particle_textcopy(e.o, entname(e), PART_TEXT, 1, 0xFF4B19, 2.0f);
        }
        loopv(ents)
        {
            entity &e = *ents[i];
            if(e.type==ET_EMPTY) continue;
            particle_textcopy(e.o, entname(e), PART_TEXT, 1, 0x1EC850, 2.0f);
            regular_particle_splash(PART_EDIT, 2, 40, e.o, 0x3232FF, 0.32f*particlesize/100.0f);
        }
    }
}
