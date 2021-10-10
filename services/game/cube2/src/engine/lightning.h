#define MAXLIGHTNINGSTEPS 64
#define LIGHTNINGSTEP 8
int lnjitterx[2][MAXLIGHTNINGSTEPS], lnjittery[2][MAXLIGHTNINGSTEPS];
int lnjitterframe = 0, lastlnjitter = 0;

VAR(lnjittermillis, 0, 100, 1000);
VAR(lnjitterradius, 0, 4, 100);
FVAR(lnjitterscale, 0, 0.5f, 10);
VAR(lnscrollmillis, 1, 300, 5000);
FVAR(lnscrollscale, 0, 0.125f, 10);
FVAR(lnblendpower, 0, 0.25f, 1000);

static void calclightningjitter(int frame)
{
    loopi(MAXLIGHTNINGSTEPS)
    {
        lnjitterx[lnjitterframe][i] = -lnjitterradius + rnd(2*lnjitterradius + 1);
        lnjittery[lnjitterframe][i] = -lnjitterradius + rnd(2*lnjitterradius + 1);
    }
}

static void setuplightning()
{
    if(!lastlnjitter || lastmillis-lastlnjitter > lnjittermillis)
    {
        if(!lastlnjitter) calclightningjitter(lnjitterframe);
        lastlnjitter = lastmillis - (lastmillis%lnjittermillis);
        calclightningjitter(lnjitterframe ^= 1);
    }
}

static void renderlightning(Texture *tex, const vec &o, const vec &d, float sz)
{
    vec step(d);
    step.sub(o);
    float len = step.magnitude();
    int numsteps = clamp(int(ceil(len/LIGHTNINGSTEP)), 2, MAXLIGHTNINGSTEPS);
    step.div(numsteps+1);
    int jitteroffset = detrnd(int(d.x+d.y+d.z), MAXLIGHTNINGSTEPS);
    vec cur(o), up, right;
    up.orthogonal(step);
    up.normalize();
    right.cross(up, step);
    right.normalize();
    float scroll = -float(lastmillis%lnscrollmillis)/lnscrollmillis, 
          scrollscale = lnscrollscale*(LIGHTNINGSTEP*tex->ys)/(sz*tex->xs),
          blend = pow(clamp(float(lastmillis - lastlnjitter)/lnjittermillis, 0.0f, 1.0f), lnblendpower),
          jitter0 = (1-blend)*lnjitterscale*sz/lnjitterradius, jitter1 = blend*lnjitterscale*sz/lnjitterradius; 
    gle::begin(GL_TRIANGLE_STRIP);
    loopj(numsteps)
    {
        vec next(cur);
        next.add(step);
        if(j+1==numsteps) next = d;
        else
        {
            int lj = (j+jitteroffset)%MAXLIGHTNINGSTEPS;
            next.add(vec(right).mul((jitter1*lnjitterx[lnjitterframe][lj] + jitter0*lnjitterx[lnjitterframe^1][lj])));
            next.add(vec(up).mul((jitter1*lnjittery[lnjitterframe][lj] + jitter0*lnjittery[lnjitterframe^1][lj])));
        }
        vec dir1 = next, dir2 = next, across;
        dir1.sub(cur);
        dir2.sub(camera1->o);
        across.cross(dir2, dir1).normalize().mul(sz);
        gle::attribf(cur.x-across.x, cur.y-across.y, cur.z-across.z);
            gle::attribf(scroll, 1);
        gle::attribf(cur.x+across.x, cur.y+across.y, cur.z+across.z);
            gle::attribf(scroll, 0);
        scroll += scrollscale;
        if(j+1==numsteps)
        {
            gle::attribf(next.x-across.x, next.y-across.y, next.z-across.z);
                gle::attribf(scroll, 1);
            gle::attribf(next.x+across.x, next.y+across.y, next.z+across.z);
                gle::attribf(scroll, 0);
        }
        cur = next;
    }
    gle::end();
}

struct lightningrenderer : listrenderer
{
    lightningrenderer()
        : listrenderer("packages/particles/lightning.jpg", 2, PT_LIGHTNING|PT_TRACK|PT_GLARE)
    {}

    void startrender()
    {
        glDisable(GL_CULL_FACE);
        gle::defattrib(gle::ATTRIB_VERTEX, 3, GL_FLOAT);
        gle::defattrib(gle::ATTRIB_TEXCOORD0, 2, GL_FLOAT);
    }

    void endrender()
    {
        glEnable(GL_CULL_FACE);
    }

    void update()
    {
        setuplightning();
    }

    void seedemitter(particleemitter &pe, const vec &o, const vec &d, int fade, float size, int gravity)
    {
        pe.maxfade = max(pe.maxfade, fade);
        pe.extendbb(o, size);
        pe.extendbb(d, size);
    }

    void renderpart(listparticle *p, const vec &o, const vec &d, int blend, int ts)
    {
        blend = min(blend<<2, 255);
        if(type&PT_MOD) //multiply alpha into color
            gle::colorub((p->color.r*blend)>>8, (p->color.g*blend)>>8, (p->color.b*blend)>>8);
        else
            gle::color(p->color, blend);
        renderlightning(tex, o, d, p->size);
    }
};
static lightningrenderer lightnings;

