#include "engine.h"

Texture *sky[6] = { 0, 0, 0, 0, 0, 0 }, *clouds[6] = { 0, 0, 0, 0, 0, 0 };

void loadsky(const char *basename, Texture *texs[6])
{
    const char *wildcard = strchr(basename, '*');
    loopi(6)
    {
        const char *side = cubemapsides[i].name;
        string name;
        copystring(name, makerelpath("packages", basename));
        if(wildcard)
        {
            char *chop = strchr(name, '*');
            if(chop) { *chop = '\0'; concatstring(name, side); concatstring(name, wildcard+1); }
            texs[i] = textureload(name, 3, true, false); 
        }
        else
        {
            defformatstring(ext, "_%s.jpg", side);
            concatstring(name, ext);
            if((texs[i] = textureload(name, 3, true, false))==notexture)
            {
                strcpy(name+strlen(name)-3, "png");
                texs[i] = textureload(name, 3, true, false);
            }
        }
        if(texs[i]==notexture) conoutf(CON_ERROR, "could not load side %s of sky texture %s", side, basename);
    }
}

Texture *cloudoverlay = NULL;

Texture *loadskyoverlay(const char *basename)
{
    const char *ext = strrchr(basename, '.'); 
    string name;
    copystring(name, makerelpath("packages", basename));
    Texture *t = notexture;
    if(ext) t = textureload(name, 0, true, false);
    else
    {
        concatstring(name, ".jpg");
        if((t = textureload(name, 0, true, false)) == notexture)
        {
            strcpy(name+strlen(name)-3, "png");
            t = textureload(name, 0, true, false);
        }
    }
    if(t==notexture) conoutf(CON_ERROR, "could not load sky overlay texture %s", basename);
    return t;
}

SVARFR(skybox, "", { if(skybox[0]) loadsky(skybox, sky); }); 
HVARR(skyboxcolour, 0, 0xFFFFFF, 0xFFFFFF);
FVARR(spinsky, -720, 0, 720);
VARR(yawsky, 0, 0, 360);
SVARFR(cloudbox, "", { if(cloudbox[0]) loadsky(cloudbox, clouds); });
HVARR(cloudboxcolour, 0, 0xFFFFFF, 0xFFFFFF);
FVARR(cloudboxalpha, 0, 1, 1);
FVARR(spinclouds, -720, 0, 720);
VARR(yawclouds, 0, 0, 360);
FVARR(cloudclip, 0, 0.5f, 1);
SVARFR(cloudlayer, "", { if(cloudlayer[0]) cloudoverlay = loadskyoverlay(cloudlayer); });
FVARR(cloudoffsetx, 0, 0, 1);
FVARR(cloudoffsety, 0, 0, 1);
FVARR(cloudscrollx, -16, 0, 16);
FVARR(cloudscrolly, -16, 0, 16);
FVARR(cloudscale, 0.001, 1, 64);
FVARR(spincloudlayer, -720, 0, 720);
VARR(yawcloudlayer, 0, 0, 360);
FVARR(cloudheight, -1, 0.2f, 1);
FVARR(cloudfade, 0, 0.2f, 1);
FVARR(cloudalpha, 0, 1, 1);
VARR(cloudsubdiv, 4, 16, 64);
HVARR(cloudcolour, 0, 0xFFFFFF, 0xFFFFFF);

void drawenvboxface(float s0, float t0, int x0, int y0, int z0,
                    float s1, float t1, int x1, int y1, int z1,
                    float s2, float t2, int x2, int y2, int z2,
                    float s3, float t3, int x3, int y3, int z3,
                    Texture *tex)
{
    glBindTexture(GL_TEXTURE_2D, (tex ? tex : notexture)->id);
    gle::begin(GL_TRIANGLE_STRIP);
    gle::attribf(x3, y3, z3); gle::attribf(s3, t3);
    gle::attribf(x2, y2, z2); gle::attribf(s2, t2);
    gle::attribf(x0, y0, z0); gle::attribf(s0, t0);
    gle::attribf(x1, y1, z1); gle::attribf(s1, t1);
    xtraverts += gle::end();
}

void drawenvbox(int w, float z1clip = 0.0f, float z2clip = 1.0f, int faces = 0x3F, Texture **sky = NULL)
{
    if(z1clip >= z2clip) return;

    float v1 = 1-z1clip, v2 = 1-z2clip;
    int z1 = int(ceil(2*w*(z1clip-0.5f))), z2 = int(ceil(2*w*(z2clip-0.5f)));

    gle::defvertex();
    gle::deftexcoord0();

    if(faces&0x01)
        drawenvboxface(0.0f, v2,  -w, -w, z2,
                       1.0f, v2,  -w,  w, z2,
                       1.0f, v1,  -w,  w, z1,
                       0.0f, v1,  -w, -w, z1, sky[0]);

    if(faces&0x02)
        drawenvboxface(1.0f, v1, w, -w, z1,
                       0.0f, v1, w,  w, z1,
                       0.0f, v2, w,  w, z2,
                       1.0f, v2, w, -w, z2, sky[1]);

    if(faces&0x04)
        drawenvboxface(1.0f, v1, -w, -w, z1,
                       0.0f, v1,  w, -w, z1,
                       0.0f, v2,  w, -w, z2,
                       1.0f, v2, -w, -w, z2, sky[2]);

    if(faces&0x08)
        drawenvboxface(1.0f, v1,  w,  w, z1,
                       0.0f, v1, -w,  w, z1,
                       0.0f, v2, -w,  w, z2,
                       1.0f, v2,  w,  w, z2, sky[3]);

    if(z1clip <= 0 && faces&0x10)
        drawenvboxface(0.0f, 1.0f, -w,  w,  -w,
                       0.0f, 0.0f,  w,  w,  -w,
                       1.0f, 0.0f,  w, -w,  -w,
                       1.0f, 1.0f, -w, -w,  -w, sky[4]);

    if(z2clip >= 1 && faces&0x20)
        drawenvboxface(0.0f, 1.0f,  w,  w, w,
                       0.0f, 0.0f, -w,  w, w,
                       1.0f, 0.0f, -w, -w, w,
                       1.0f, 1.0f,  w, -w, w, sky[5]);
}

void drawenvoverlay(int w, Texture *overlay = NULL, float tx = 0, float ty = 0)
{
    float z = w*cloudheight, tsz = 0.5f*(1-cloudfade)/cloudscale, psz = w*(1-cloudfade);
    glBindTexture(GL_TEXTURE_2D, overlay ? overlay->id : notexture->id);
    vec color = vec::hexcolor(cloudcolour);
    gle::color(color, cloudalpha);
    gle::defvertex();
    gle::deftexcoord0();
    gle::begin(GL_TRIANGLE_FAN);
    loopi(cloudsubdiv+1)
    {
        vec p(1, 1, 0);
        p.rotate_around_z((-2.0f*M_PI*i)/cloudsubdiv);
        gle::attribf(p.x*psz, p.y*psz, z);
            gle::attribf(tx + p.x*tsz, ty + p.y*tsz);
    }
    xtraverts += gle::end();
    float tsz2 = 0.5f/cloudscale;
    gle::defvertex();
    gle::deftexcoord0();
    gle::defcolor(4);
    gle::begin(GL_TRIANGLE_STRIP);
    loopi(cloudsubdiv+1)
    {
        vec p(1, 1, 0);
        p.rotate_around_z((-2.0f*M_PI*i)/cloudsubdiv);
        gle::attribf(p.x*psz, p.y*psz, z);
            gle::attribf(tx + p.x*tsz, ty + p.y*tsz);
            gle::attrib(color, cloudalpha);
        gle::attribf(p.x*w, p.y*w, z);
            gle::attribf(tx + p.x*tsz2, ty + p.y*tsz2);
            gle::attrib(color, 0.0f);
    }
    xtraverts += gle::end();
}

FVARR(fogdomeheight, -1, -0.5f, 1);
FVARR(fogdomemin, 0, 0, 1);
FVARR(fogdomemax, 0, 0, 1);
VARR(fogdomecap, 0, 1, 1);
FVARR(fogdomeclip, 0, 1, 1);
bvec fogdomecolor(0, 0, 0);
HVARFR(fogdomecolour, 0, 0, 0xFFFFFF,
{
    fogdomecolor = bvec((fogdomecolour>>16)&0xFF, (fogdomecolour>>8)&0xFF, fogdomecolour&0xFF);
});
VARR(fogdomeclouds, 0, 1, 1);

namespace fogdome
{
    struct vert
    {
        vec pos;
        bvec4 color;
    
    	vert() {}
    	vert(const vec &pos, const bvec &fcolor, float alpha) : pos(pos), color(fcolor, uchar(alpha*255))
    	{
    	}
        vert(const vert &v0, const vert &v1) : pos(vec(v0.pos).add(v1.pos).normalize()), color(v0.color)
    	{
            if(v0.pos.z != v1.pos.z) color.a += uchar((v1.color.a - v0.color.a) * (pos.z - v0.pos.z) / (v1.pos.z - v0.pos.z));
    	}
    } *verts = NULL;
    GLushort *indices = NULL;
    int numverts = 0, numindices = 0, capindices = 0;
    GLuint vbuf = 0, ebuf = 0;
    bvec lastcolor(0, 0, 0);
    float lastminalpha = 0, lastmaxalpha = 0, lastcapsize = -1, lastclipz = 1;
    
    void subdivide(int depth, int face);
    
    void genface(int depth, int i1, int i2, int i3)
    {
        int face = numindices; numindices += 3;
        indices[face]   = i3;
        indices[face+1] = i2;
        indices[face+2] = i1;
        subdivide(depth, face);
    }
    
    void subdivide(int depth, int face)
    {
        if(depth-- <= 0) return;
        int idx[6];
        loopi(3) idx[i] = indices[face+2-i];
        loopi(3)
        {
            int curvert = numverts++;
            verts[curvert] = vert(verts[idx[i]], verts[idx[(i+1)%3]]); //push on to unit sphere
            idx[3+i] = curvert;
            indices[face+2-i] = curvert;
        }
        subdivide(depth, face);
        loopi(3) genface(depth, idx[i], idx[3+i], idx[3+(i+2)%3]);
    }
    
    int sortcap(GLushort x, GLushort y)
    {
        const vec &xv = verts[x].pos, &yv = verts[y].pos;
        return xv.y < 0 ? yv.y >= 0 || xv.x < yv.x : yv.y >= 0 && xv.x > yv.x;
    }
    
    void init(const bvec &color, float minalpha = 0.0f, float maxalpha = 1.0f, float capsize = -1, float clipz = 1, int hres = 16, int depth = 2)
    {
        const int tris = hres << (2*depth);
        numverts = numindices = capindices = 0;
        verts = new vert[tris+1 + (capsize >= 0 ? 1 : 0)];
        indices = new GLushort[(tris + (capsize >= 0 ? hres<<depth : 0))*3];
        if(clipz >= 1)
        {
            verts[numverts++] = vert(vec(0.0f, 0.0f, 1.0f), color, minalpha); //build initial 'hres' sided pyramid
            loopi(hres) verts[numverts++] = vert(vec(sincos360[(360*i)/hres], 0.0f), color, maxalpha);
            loopi(hres) genface(depth, 0, i+1, 1+(i+1)%hres);
        }
        else if(clipz <= 0)
        {
            loopi(hres<<depth) verts[numverts++] = vert(vec(sincos360[(360*i)/(hres<<depth)], 0.0f), color, maxalpha);
        }
        else
        {
            float clipxy = sqrtf(1 - clipz*clipz);
            const vec2 &scm = sincos360[180/hres];
            loopi(hres)
            {
                const vec2 &sc = sincos360[(360*i)/hres];
                verts[numverts++] = vert(vec(sc.x*clipxy, sc.y*clipxy, clipz), color, minalpha);
                verts[numverts++] = vert(vec(sc.x, sc.y, 0.0f), color, maxalpha);
                verts[numverts++] = vert(vec(sc.x*scm.x - sc.y*scm.y, sc.y*scm.x + sc.x*scm.y, 0.0f), color, maxalpha);
            }
            loopi(hres)
            {
                genface(depth-1, 3*i, 3*i+1, 3*i+2);
                genface(depth-1, 3*i, 3*i+2, 3*((i+1)%hres));
                genface(depth-1, 3*i+2, 3*((i+1)%hres)+1, 3*((i+1)%hres));
            }
        }
    
        if(capsize >= 0)
        {
            GLushort *cap = &indices[numindices];
            int capverts = 0;
            loopi(numverts) if(!verts[i].pos.z) cap[capverts++] = i;
            verts[numverts++] = vert(vec(0.0f, 0.0f, -capsize), color, maxalpha);
            quicksort(cap, capverts, sortcap); 
            loopi(capverts)
            {
                int n = capverts-1-i;
                cap[n*3] = cap[n];
                cap[n*3+1] = cap[(n+1)%capverts];
                cap[n*3+2] = numverts-1;
                capindices += 3;
            }
        }
    
        if(!vbuf) glGenBuffers_(1, &vbuf);
        gle::bindvbo(vbuf);
        glBufferData_(GL_ARRAY_BUFFER, numverts*sizeof(vert), verts, GL_STATIC_DRAW);
        DELETEA(verts);
    
        if(!ebuf) glGenBuffers_(1, &ebuf);
        gle::bindebo(ebuf);
        glBufferData_(GL_ELEMENT_ARRAY_BUFFER, (numindices + capindices)*sizeof(GLushort), indices, GL_STATIC_DRAW);
        DELETEA(indices);
    }
    
    void cleanup()
    {
    	numverts = numindices = 0;
        if(vbuf) { glDeleteBuffers_(1, &vbuf); vbuf = 0; }
        if(ebuf) { glDeleteBuffers_(1, &ebuf); ebuf = 0; }
    }
    
    void draw()
    {
        float capsize = fogdomecap && fogdomeheight < 1 ? (1 + fogdomeheight) / (1 - fogdomeheight) : -1;
        bvec color = fogdomecolour ? fogdomecolor : fogcolor;
        if(!numverts || lastcolor != color || lastminalpha != fogdomemin || lastmaxalpha != fogdomemax || lastcapsize != capsize || lastclipz != fogdomeclip)
        {
            init(color, min(fogdomemin, fogdomemax), fogdomemax, capsize, fogdomeclip);
            lastcolor = color;
            lastminalpha = fogdomemin;
            lastmaxalpha = fogdomemax;
            lastcapsize = capsize;
            lastclipz = fogdomeclip;
        }
    
        gle::bindvbo(vbuf);
        gle::bindebo(ebuf);

        gle::vertexpointer(sizeof(vert), &verts->pos);
        gle::colorpointer(sizeof(vert), &verts->color);
        gle::enablevertex();
        gle::enablecolor();
    
        glDrawRangeElements_(GL_TRIANGLES, 0, numverts-1, numindices + fogdomecap*capindices, GL_UNSIGNED_SHORT, indices);
        xtraverts += numverts;
        glde++;

        gle::disablevertex();
        gle::disablecolor();
    
        gle::clearvbo();
        gle::clearebo();
    }
}

static void drawfogdome(int farplane)
{
    SETSHADER(skyfog);

    glEnable(GL_BLEND);
    glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

    matrix4 skymatrix = cammatrix, skyprojmatrix;
    skymatrix.settranslation(vec(cammatrix.c).mul(farplane*fogdomeheight*0.5f));
    skymatrix.scale(farplane/2, farplane/2, farplane*(0.5f - fogdomeheight*0.5f));
    skyprojmatrix.mul(projmatrix, skymatrix);
    LOCALPARAM(skymatrix, skyprojmatrix);

    fogdome::draw();

    glDisable(GL_BLEND);
}

void cleanupsky()
{
    fogdome::cleanup();
}

extern int atmo;

void preloadatmoshaders(bool force = false)
{
    static bool needatmo = false;
    if(force) needatmo = true;
    if(!atmo || !needatmo) return;

    useshaderbyname("atmosphere");
    useshaderbyname("atmosphereglare");
}

void setupsky()
{
    preloadatmoshaders(true);
}

VARFR(atmo, 0, 0, 1, preloadatmoshaders());
FVARR(atmoplanetsize, 1e-3f, 1, 1e3f);
FVARR(atmoheight, 1e-3f, 1, 1e3f);
FVARR(atmobright, 0, 1, 16);
bvec atmosunlightcolor(0, 0, 0);
HVARFR(atmosunlight, 0, 0, 0xFFFFFF,
{
    if(atmosunlight <= 255) atmosunlight |= (atmosunlight<<8) | (atmosunlight<<16);
    atmosunlightcolor = bvec((atmosunlight>>16)&0xFF, (atmosunlight>>8)&0xFF, atmosunlight&0xFF);
});
FVARR(atmosunlightscale, 0, 1, 16);
bvec atmosundiskcolor(0, 0, 0);
HVARFR(atmosundisk, 0, 0, 0xFFFFFF,
{
    if(atmosundisk <= 255) atmosundisk |= (atmosundisk<<8) | (atmosundisk<<16);
    atmosundiskcolor = bvec((atmosundisk>>16)&0xFF, (atmosundisk>>8)&0xFF, atmosundisk&0xFF);
});
FVARR(atmosundisksize, 0, 12, 90);
FVARR(atmosundiskcorona, 0, 0.4f, 1);
FVARR(atmosundiskbright, 0, 1, 16);
FVARR(atmohaze, 0, 0.1f, 16);
FVARR(atmodensity, 0, 1, 16);
FVARR(atmoozone, 0, 1, 16);
FVARR(atmoalpha, 0, 1, 1);

static void drawatmosphere(int w, float z1clip = 0.0f, float z2clip = 1.0f, int faces = 0x3F)
{
    if(z1clip >= z2clip) return;

    if(glaring) SETSHADER(atmosphereglare);
    else SETSHADER(atmosphere);

    matrix4 skymatrix = cammatrix, skyprojmatrix;
    skymatrix.settranslation(0, 0, 0);
    skyprojmatrix.mul(projmatrix, skymatrix);
    LOCALPARAM(skymatrix, skyprojmatrix);

    // optical depth scales for 3 different shells of atmosphere - air, haze, ozone
    const float earthradius = 6371e3f, earthairheight = 8.4e3f, earthhazeheight = 1.25e3f, earthozoneheight = 50e3f;
    float planetradius = earthradius*atmoplanetsize;
    vec atmoshells = vec(earthairheight, earthhazeheight, earthozoneheight).mul(atmoheight).add(planetradius).square().sub(planetradius*planetradius);
    LOCALPARAM(opticaldepthparams, vec4(atmoshells, planetradius));

    // Henyey-Greenstein approximation, 1/(4pi) * (1 - g^2)/(1 + g^2 - 2gcos)]^1.5
    // Hoffman-Preetham variation uses (1-g)^2 instead of 1-g^2 which avoids excessive glare
    // clamp values near 0 angle to avoid spotlight artifact inside sundisk
    float gm = max(0.95f - 0.2f*atmohaze, 0.65f), miescale = pow((1-gm)*(1-gm)/(4*M_PI), -2.0f/3.0f);
    LOCALPARAMF(mieparams, miescale*(1 + gm*gm), miescale*-2*gm, 1 - (1 - cosf(0.5f*atmosundisksize*(1 - atmosundiskcorona)*RAD)));

    static const vec lambda(680e-9f, 550e-9f, 450e-9f),
                     k(0.686f, 0.678f, 0.666f),
                     ozone(3.426f, 8.298f, 0.356f);
    vec betar = vec(lambda).square().square().recip().mul(1.241e-30f/M_LN2 * atmodensity),
        betam = vec(lambda).recip().square().mul(k).mul(9.072e-17f/M_LN2 * atmohaze),
        betao = vec(ozone).mul(1.5e-7f/M_LN2 * atmoozone);
    LOCALPARAM(betarayleigh, betar);
    LOCALPARAM(betamie, betam);
    LOCALPARAM(betaozone, betao);

    // extinction in direction of sun
    float sunoffset = sunlightdir.z*planetradius;
    vec sundepth = vec(atmoshells).add(sunoffset*sunoffset).sqrt().sub(sunoffset);
    vec sunweight = vec(betar).mul(sundepth.x).madd(betam, sundepth.y).madd(betao, sundepth.z - sundepth.x);
    vec sunextinction = vec(sunweight).neg().exp2();
    vec suncolor = atmosunlight ? atmosunlightcolor.tocolor().mul(atmosunlightscale) : sunlightcolor.tocolor().mul(sunlightscale);
    // assume sunlight color is gamma encoded, so decode to linear light, then apply extinction
    vec sunscale = vec(suncolor).square().mul(atmobright * 16).mul(sunextinction);
    float maxsunweight = max(max(sunweight.x, sunweight.y), sunweight.z);
    if(maxsunweight > 127) sunweight.mul(127/maxsunweight);
    sunweight.add(1e-4f);
    LOCALPARAM(sunweight, sunweight);
    LOCALPARAM(sunlight, vec4(sunscale, atmoalpha));
    LOCALPARAM(sundir, sunlightdir);

    // invert extinction at zenith to get an approximation of how bright the sun disk should be
    vec zenithdepth = vec(atmoshells).add(planetradius*planetradius).sqrt().sub(planetradius);
    vec zenithweight = vec(betar).mul(zenithdepth.x).madd(betam, zenithdepth.y).madd(betao, zenithdepth.z - zenithdepth.x);
    vec zenithextinction = vec(zenithweight).sub(sunweight).exp2();
    vec diskcolor = (atmosundisk ? atmosundiskcolor.tocolor() : suncolor).square().mul(zenithextinction).mul(atmosundiskbright * (glaring ? 1 : 1.5f)).min(1);
    LOCALPARAM(sundiskcolor, diskcolor);

    // convert from view cosine into mu^2 for limb darkening, where mu = sqrt(1 - sin^2) and sin^2 = 1 - cos^2, thus mu^2 = 1 - (1 - cos^2*scale)
    // convert corona offset into scale for mu^2, where sin = (1-corona) and thus mu^2 = 1 - (1-corona^2) 
    float sundiskscale = sinf(0.5f*atmosundisksize*RAD);
    float coronamu = 1 - (1-atmosundiskcorona)*(1-atmosundiskcorona);
    if(sundiskscale > 0) LOCALPARAMF(sundiskparams, 1.0f/(sundiskscale*sundiskscale), 1.0f/max(coronamu, 1e-3f));
    else LOCALPARAMF(sundiskparams, 0, 0);

    float z1 = 2*w*(z1clip-0.5f), z2 = ceil(2*w*(z2clip-0.5f));

    gle::defvertex();

    if(glaring)
    {
        if(sundiskscale > 0 && sunlightdir.z*w + sundiskscale > z1 && sunlightdir.z*w - sundiskscale < z2)
        {
            gle::begin(GL_TRIANGLE_FAN);
            vec spoke;
            spoke.orthogonal(sunlightdir);
            spoke.rescale(2*sundiskscale);
            loopi(4) gle::attrib(vec(spoke).rotate(-2*M_PI*i/4.0f, sunlightdir).add(sunlightdir).mul(w));
            xtraverts += gle::end();
        }
        return;
    }

    gle::begin(GL_QUADS);

    if(faces&0x01)
    {
        gle::attribf(-w, -w, z1);
        gle::attribf(-w,  w, z1);
        gle::attribf(-w,  w, z2);
        gle::attribf(-w, -w, z2);
    }

    if(faces&0x02)
    {
        gle::attribf(w, -w, z2);
        gle::attribf(w,  w, z2);
        gle::attribf(w,  w, z1);
        gle::attribf(w, -w, z1);
    }

    if(faces&0x04)
    {
        gle::attribf(-w, -w, z2);
        gle::attribf( w, -w, z2);
        gle::attribf( w, -w, z1);
        gle::attribf(-w, -w, z1);
    }

    if(faces&0x08)
    {
        gle::attribf( w, w, z2);
        gle::attribf(-w, w, z2);
        gle::attribf(-w, w, z1);
        gle::attribf( w, w, z1);
    }

    if(z1clip <= 0 && faces&0x10)
    {
        gle::attribf(-w, -w, -w);
        gle::attribf( w, -w, -w);
        gle::attribf( w,  w, -w);
        gle::attribf(-w,  w, -w);
    }

    if(z2clip >= 1 && faces&0x20)
    {
        gle::attribf( w, -w, w);
        gle::attribf(-w, -w, w);
        gle::attribf(-w,  w, w);
        gle::attribf( w,  w, w);
    }

    xtraverts += gle::end();
}

VARP(sparklyfix, 0, 0, 1);
VAR(showsky, 0, 1, 1); 
VAR(clipsky, 0, 1, 1);

bool drawskylimits(bool explicitonly)
{
    nocolorshader->set();

    glColorMask(GL_FALSE, GL_FALSE, GL_FALSE, GL_FALSE);
    bool rendered = rendersky(explicitonly);
    glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_TRUE);

    return rendered;
}

void drawskyoutline()
{
    notextureshader->set();

    glDepthMask(GL_FALSE);
    extern int wireframe;
    if(!wireframe)
    {
        enablepolygonoffset(GL_POLYGON_OFFSET_LINE);
        glPolygonMode(GL_FRONT_AND_BACK, GL_LINE);
    }
    gle::colorf(0.5f, 0.0f, 0.5f);
    rendersky(true);
    if(!wireframe) 
    {
        glPolygonMode(GL_FRONT_AND_BACK, GL_FILL);
        disablepolygonoffset(GL_POLYGON_OFFSET_LINE);
    }
    glDepthMask(GL_TRUE);
}

VAR(clampsky, 0, 1, 1);

static int yawskyfaces(int faces, int yaw, float spin = 0)
{
    if(spin || yaw%90) return faces&0x0F ? faces | 0x0F : faces;
    static const int faceidxs[3][4] =
    {
        { 3, 2, 0, 1 },
        { 1, 0, 3, 2 },
        { 2, 3, 1, 0 }
    };
    yaw /= 90;
    if(yaw < 1 || yaw > 3) return faces;
    const int *idxs = faceidxs[yaw - 1];
    return (faces & ~0x0F) | (((faces>>idxs[0])&1)<<0) | (((faces>>idxs[1])&1)<<1) | (((faces>>idxs[2])&1)<<2) | (((faces>>idxs[3])&1)<<3);
}

void drawskybox(int farplane, bool limited, bool force)
{
    extern int renderedskyfaces, renderedskyclip; // , renderedsky, renderedexplicitsky;
    bool alwaysrender = editmode || !insideworld(camera1->o) || reflecting || force,
         explicitonly = false;
    if(limited)
    {
        explicitonly = alwaysrender || !sparklyfix || refracting; 
        if(!drawskylimits(explicitonly) && !alwaysrender) return;
        extern int ati_skybox_bug;
        if(!alwaysrender && !renderedskyfaces && !ati_skybox_bug) explicitonly = false;
    }
    else if(!alwaysrender)
    {
        renderedskyfaces = 0;
        renderedskyclip = INT_MAX;
        for(vtxarray *va = visibleva; va; va = va->next)
        {
            if(va->occluded >= OCCLUDE_BB && va->skyfaces&0x80) continue;
            renderedskyfaces |= va->skyfaces&0x3F;
            if(!(va->skyfaces&0x1F) || camera1->o.z < va->skyclip) renderedskyclip = min(renderedskyclip, va->skyclip);
            else renderedskyclip = 0;
        }
        if(!renderedskyfaces) return;
    }
    
    if(alwaysrender)
    {
        renderedskyfaces = 0x3F;
        renderedskyclip = 0;
    }

    float skyclip = clipsky ? max(renderedskyclip-1, 0) : 0, topclip = 1;
    if(reflectz<worldsize)
    {
        if(refracting<0) topclip = 0.5f + 0.5f*(reflectz-camera1->o.z)/float(worldsize);
        else if(reflectz>skyclip) skyclip = reflectz;
    }
    if(skyclip) skyclip = 0.5f + 0.5f*(skyclip-camera1->o.z)/float(worldsize); 

    if(limited) 
    {
        if(explicitonly) glDisable(GL_DEPTH_TEST);
        else glDepthFunc(GL_GEQUAL);
    }
    else glDepthFunc(GL_LEQUAL);

    glDepthMask(GL_FALSE);

    if(clampsky) glDepthRange(1, 1);

    if(!atmo || (skybox[0] && atmoalpha < 1))
    {
        if(glaring) SETSHADER(skyboxglare);
        else SETSHADER(skybox);

        gle::color(vec::hexcolor(skyboxcolour));

        matrix4 skymatrix = cammatrix, skyprojmatrix;
        skymatrix.settranslation(0, 0, 0);
        skymatrix.rotate_around_z((spinsky*lastmillis/1000.0f+yawsky)*-RAD);
        skyprojmatrix.mul(projmatrix, skymatrix);
        LOCALPARAM(skymatrix, skyprojmatrix);

        drawenvbox(farplane/2, skyclip, topclip, yawskyfaces(renderedskyfaces, yawsky, spinsky), sky);
    }

    if(atmo)
    {
        if(atmoalpha < 1)
        {
            if(fading) glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_FALSE);
            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
        }

        drawatmosphere(farplane/2, skyclip, topclip, renderedskyfaces);

        if(atmoalpha < 1) glDisable(GL_BLEND);
    }

    if(!glaring)
    {
        if(fogdomemax && !fogdomeclouds)
        {
            if(fading) glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_FALSE);
            drawfogdome(farplane);
        }

        if(cloudbox[0])
        {
            SETSHADER(skybox);

            if(fading) glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_FALSE);

            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

            gle::color(vec::hexcolor(cloudboxcolour), cloudboxalpha);

            matrix4 skymatrix = cammatrix, skyprojmatrix;
            skymatrix.settranslation(0, 0, 0);
            skymatrix.rotate_around_z((spinclouds*lastmillis/1000.0f+yawclouds)*-RAD);
            skyprojmatrix.mul(projmatrix, skymatrix);
            LOCALPARAM(skymatrix, skyprojmatrix);

            drawenvbox(farplane/2, skyclip ? skyclip : cloudclip, topclip, yawskyfaces(renderedskyfaces, yawclouds, spinclouds), clouds);

            glDisable(GL_BLEND);
        }

        if(cloudlayer[0] && cloudheight && renderedskyfaces&(cloudheight < 0 ? 0x1F : 0x2F))
        {
            SETSHADER(skybox);

            if(fading) glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_FALSE);

            glDisable(GL_CULL_FACE);

            glEnable(GL_BLEND);
            glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);

            matrix4 skymatrix = cammatrix, skyprojmatrix;
            skymatrix.settranslation(0, 0, 0);
            skymatrix.rotate_around_z((spincloudlayer*lastmillis/1000.0f+yawcloudlayer)*-RAD);
            skyprojmatrix.mul(projmatrix, skymatrix);
            LOCALPARAM(skymatrix, skyprojmatrix);

            drawenvoverlay(farplane/2, cloudoverlay, cloudoffsetx + cloudscrollx * lastmillis/1000.0f, cloudoffsety + cloudscrolly * lastmillis/1000.0f);

            glDisable(GL_BLEND);

            glEnable(GL_CULL_FACE);
        }

	    if(fogdomemax && fogdomeclouds)
	    {
            if(fading) glColorMask(GL_TRUE, GL_TRUE, GL_TRUE, GL_FALSE);
            drawfogdome(farplane);
	    }
    }

    if(clampsky) glDepthRange(0, 1);

    glDepthMask(GL_TRUE);

    if(limited)
    {
        if(explicitonly) glEnable(GL_DEPTH_TEST);
        else glDepthFunc(GL_LESS);
        if(!reflecting && !refracting && !drawtex && editmode && showsky) drawskyoutline();
    }
    else glDepthFunc(GL_LESS);
}

VARNR(skytexture, useskytexture, 0, 1, 1);

int explicitsky = 0;
double skyarea = 0;

bool limitsky()
{
    return (explicitsky && (useskytexture || editmode)) || (sparklyfix && skyarea / (double(worldsize)*double(worldsize)*6) < 0.9);
}

bool shouldrenderskyenvmap()
{
    return atmo != 0;
}

bool shouldclearskyboxglare()
{
    return atmo && (!skybox[0] || atmoalpha >= 1);
}

