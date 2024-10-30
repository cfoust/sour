#include "engine.h"
#include "rendertarget.h"

static struct glaretexture : rendertarget
{
    bool dorender()
    {
        extern void drawglare();
        drawglare();
        return true;
    }
} glaretex;

void cleanupglare()
{
    glaretex.cleanup(true);
}

VARFP(glaresize, 6, 8, 10, cleanupglare());
VARP(glare, 0, 0, 1);
VARP(blurglare, 0, 4, 7);
VARP(blurglareaspect, 0, 1, 1);
VARP(blurglaresigma, 1, 50, 200);

VAR(debugglare, 0, 0, 1);

void viewglaretex()
{
    if(!glare) return;
    glaretex.debug();
}

bool glaring = false;

void drawglaretex()
{
    if(!glare) return;

    int w = 1<<glaresize, h = 1<<glaresize, blury = blurglare;
    if(blurglare && blurglareaspect)
    {
        while(h > (1<<5) && (screenw*h)/w >= (screenh*4)/3) h /= 2;
        blury = ((1 + 4*blurglare)*(screenw*h)/w + screenh*2)/(screenh*4);
        blury = clamp(blury, 1, MAXBLURRADIUS);
    }

    glaretex.render(w, h, blurglare, blurglaresigma/100.0f, blury);
}

FVAR(glaremod, 0.5f, 0.75f, 1);
FVARP(glarescale, 0, 1, 8);

void addglare()
{
    if(!glare) return;

    glEnable(GL_BLEND);
    glBlendFunc(GL_ONE, GL_ONE);

    SETSHADER(screenrect);

    glBindTexture(GL_TEXTURE_2D, glaretex.rendertex);

    float g = glarescale*glaremod;
    gle::colorf(g, g, g);

    screenquad(1, 1);

    glDisable(GL_BLEND);
}
     
