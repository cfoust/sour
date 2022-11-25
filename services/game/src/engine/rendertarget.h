extern int rtsharefb, rtscissor, blurtile;

struct rendertarget
{
    int texw, texh, vieww, viewh;
    GLenum colorfmt, depthfmt;
    GLuint rendertex, renderfb, renderdb, blurtex, blurfb, blurdb;
    int blursize, blurysize;
    float blursigma;
    float blurweights[MAXBLURRADIUS+1], bluroffsets[MAXBLURRADIUS+1];
    float bluryweights[MAXBLURRADIUS+1], bluryoffsets[MAXBLURRADIUS+1];

    float scissorx1, scissory1, scissorx2, scissory2;
#define BLURTILES 32
#define BLURTILEMASK (0xFFFFFFFFU>>(32-BLURTILES))
    uint blurtiles[BLURTILES+1];

    bool initialized;

    rendertarget() : texw(0), texh(0), vieww(0), viewh(0), colorfmt(GL_FALSE), depthfmt(GL_FALSE), rendertex(0), renderfb(0), renderdb(0), blurtex(0), blurfb(0), blurdb(0), blursize(0), blurysize(0), blursigma(0), initialized(false)
    {
    }

    virtual ~rendertarget() {}

    virtual GLenum attachment() const
    {
        return GL_COLOR_ATTACHMENT0;
    }

    virtual const GLenum *colorformats() const
    {
        static const GLenum colorfmts[] = { GL_RGB, GL_RGB8, GL_FALSE };
        return colorfmts;
    }

    virtual const GLenum *depthformats() const
    {
        static const GLenum depthfmts[] = { GL_DEPTH_COMPONENT24, GL_DEPTH_COMPONENT, GL_DEPTH_COMPONENT16, GL_DEPTH_COMPONENT32, GL_FALSE };
        return depthfmts;
    }

    virtual bool depthtest() const { return true; }

    void cleanup(bool fullclean = false)
    {
        if(renderfb) { glDeleteFramebuffers_(1, &renderfb); renderfb = 0; }
        if(renderdb) { glDeleteRenderbuffers_(1, &renderdb); renderdb = 0; }
        if(rendertex) { glDeleteTextures(1, &rendertex); rendertex = 0; }
        texw = texh = 0;
        cleanupblur();

        if(fullclean) colorfmt = depthfmt = GL_FALSE;
    }

    void cleanupblur()
    {
        if(blurfb) { glDeleteFramebuffers_(1, &blurfb); blurfb = 0; }
        if(blurtex) { glDeleteTextures(1, &blurtex); blurtex = 0; }
        if(blurdb) { glDeleteRenderbuffers_(1, &blurdb); blurdb = 0; }
        blursize = blurysize = 0;
        blursigma = 0.0f;
    }

    void setupblur()
    {
        if(!blurtex) glGenTextures(1, &blurtex);
        createtexture(blurtex, texw, texh, NULL, 3, 1, colorfmt);

        if(!swaptexs() || rtsharefb) return;
        if(!blurfb) glGenFramebuffers_(1, &blurfb);
        glBindFramebuffer_(GL_FRAMEBUFFER, blurfb);
        glFramebufferTexture2D_(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, blurtex, 0);
        if(depthtest())
        {
            if(!blurdb) glGenRenderbuffers_(1, &blurdb);
            glGenRenderbuffers_(1, &blurdb);
            glBindRenderbuffer_(GL_RENDERBUFFER, blurdb);
            glRenderbufferStorage_(GL_RENDERBUFFER, depthfmt, texw, texh);
            glFramebufferRenderbuffer_(GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, blurdb);
        }
        glBindFramebuffer_(GL_FRAMEBUFFER, 0);
    }

    void setup(int w, int h)
    {
        if(!renderfb) glGenFramebuffers_(1, &renderfb);
        glBindFramebuffer_(GL_FRAMEBUFFER, renderfb);
        if(!rendertex) glGenTextures(1, &rendertex);

        GLenum attach = attachment();
        if(attach == GL_DEPTH_ATTACHMENT)
        {
            glDrawBuffer(GL_NONE);
            glReadBuffer(GL_NONE);
        }

        const GLenum *colorfmts = colorformats();
        int find = 0;
        do
        {
            createtexture(rendertex, w, h, NULL, 3, filter() ? 1 : 0, colorfmt ? colorfmt : colorfmts[find]);
            glFramebufferTexture2D_(GL_FRAMEBUFFER, attach, GL_TEXTURE_2D, rendertex, 0);
            if(glCheckFramebufferStatus_(GL_FRAMEBUFFER)==GL_FRAMEBUFFER_COMPLETE) break;
        }
        while(!colorfmt && colorfmts[++find]);
        if(!colorfmt) colorfmt = colorfmts[find];

        if(attach != GL_DEPTH_ATTACHMENT && depthtest())
        {
            if(!renderdb) { glGenRenderbuffers_(1, &renderdb); depthfmt = GL_FALSE; }
            if(!depthfmt) glBindRenderbuffer_(GL_RENDERBUFFER, renderdb);
            const GLenum *depthfmts = depthformats();
            find = 0;
            do
            {
                if(!depthfmt) glRenderbufferStorage_(GL_RENDERBUFFER, depthfmts[find], w, h);
                glFramebufferRenderbuffer_(GL_FRAMEBUFFER, GL_DEPTH_ATTACHMENT, GL_RENDERBUFFER, renderdb);
                if(glCheckFramebufferStatus_(GL_FRAMEBUFFER)==GL_FRAMEBUFFER_COMPLETE) break;
            }
            while(!depthfmt && depthfmts[++find]);
            if(!depthfmt) depthfmt = depthfmts[find];
        }
 
        glBindFramebuffer_(GL_FRAMEBUFFER, 0);

        texw = w;
        texh = h;
        initialized = false;
    }

    bool addblurtiles(float x1, float y1, float x2, float y2, float blurmargin = 0)
    {
        if(x1 >= 1 || y1 >= 1 || x2 <= -1 || y2 <= -1) return false;

        scissorx1 = min(scissorx1, max(x1, -1.0f));
        scissory1 = min(scissory1, max(y1, -1.0f));
        scissorx2 = max(scissorx2, min(x2, 1.0f));
        scissory2 = max(scissory2, min(y2, 1.0f));

        float blurerror = 2.0f*float(2*blursize + blurmargin);
        int tx1 = max(0, min(BLURTILES - 1, int((x1-blurerror/vieww + 1)/2 * BLURTILES))),
            ty1 = max(0, min(BLURTILES - 1, int((y1-blurerror/viewh + 1)/2 * BLURTILES))),
            tx2 = max(0, min(BLURTILES - 1, int((x2+blurerror/vieww + 1)/2 * BLURTILES))),
            ty2 = max(0, min(BLURTILES - 1, int((y2+blurerror/viewh + 1)/2 * BLURTILES)));

        uint mask = (BLURTILEMASK>>(BLURTILES - (tx2+1))) & (BLURTILEMASK<<tx1);
        for(int y = ty1; y <= ty2; y++) blurtiles[y] |= mask;
        return true;
    }

    bool checkblurtiles(float x1, float y1, float x2, float y2, float blurmargin = 0)
    {
        float blurerror = 2.0f*float(2*blursize + blurmargin);
        if(x2+blurerror/vieww < scissorx1 || y2+blurerror/viewh < scissory1 || 
           x1-blurerror/vieww > scissorx2 || y1-blurerror/viewh > scissory2) 
            return false;

        if(!blurtile) return true;

        int tx1 = max(0, min(BLURTILES - 1, int((x1 + 1)/2 * BLURTILES))),
            ty1 = max(0, min(BLURTILES - 1, int((y1 + 1)/2 * BLURTILES))),
            tx2 = max(0, min(BLURTILES - 1, int((x2 + 1)/2 * BLURTILES))),
            ty2 = max(0, min(BLURTILES - 1, int((y2 + 1)/2 * BLURTILES)));

        uint mask = (BLURTILEMASK>>(BLURTILES - (tx2+1))) & (BLURTILEMASK<<tx1);
        for(int y = ty1; y <= ty2; y++) if(blurtiles[y] & mask) return true;

        return false;
    }

    void rendertiles()
    {
        float wscale = vieww/float(texw), hscale = viewh/float(texh);
        if(blurtile && scissorx1 < scissorx2 && scissory1 < scissory2)
        {
            uint tiles[sizeof(blurtiles)/sizeof(uint)];
            memcpy(tiles, blurtiles, sizeof(blurtiles));

            LOCALPARAMF(screentexcoord0, wscale*0.5f, hscale*0.5f, wscale*0.5f, hscale*0.5f);
            gle::defvertex(2);
            gle::begin(GL_QUADS);
            float tsz = 1.0f/BLURTILES;
            loop(y, BLURTILES+1)
            {
                uint mask = tiles[y];
                int x = 0;
                while(mask)
                {
                    while(!(mask&0xFF)) { mask >>= 8; x += 8; }
                    while(!(mask&1)) { mask >>= 1; x++; }
                    int xstart = x;
                    do { mask >>= 1; x++; } while(mask&1);
                    uint strip = (BLURTILEMASK>>(BLURTILES - x)) & (BLURTILEMASK<<xstart);
                    int yend = y;
                    do { tiles[yend] &= ~strip; yend++; } while((tiles[yend] & strip) == strip);
                    float tx = xstart*tsz,
                          ty = y*tsz,
                          tw = (x-xstart)*tsz,
                          th = (yend-y)*tsz,
                          vx = 2*tx - 1, vy = 2*ty - 1, vw = tw*2, vh = th*2;
                    gle::attribf(vx,    vy);
                    gle::attribf(vx+vw, vy);
                    gle::attribf(vx+vw, vy+vh);
                    gle::attribf(vx,    vy+vh);
                }
            }
            gle::end();
        }
        else
        {
            screenquad(wscale, hscale);
        }
    }

    void blur(int wantsblursize, float wantsblursigma, int wantsblurysize, int x, int y, int w, int h, bool scissor)
    {
        if(!blurtex) setupblur();
        if(blursize!=wantsblursize || blurysize != wantsblurysize || (wantsblursize && blursigma!=wantsblursigma))
        {
            setupblurkernel(wantsblursize, wantsblursigma, blurweights, bluroffsets);
            if(wantsblurysize != wantsblursize) setupblurkernel(wantsblurysize, wantsblursigma, bluryweights, bluryoffsets);
            blursize = wantsblursize;
            blursigma = wantsblursigma;
            blurysize = wantsblurysize;
        }

        glDisable(GL_DEPTH_TEST);
        glDisable(GL_CULL_FACE);

        if(scissor)
        {
            glScissor(x, y, w, h);
            glEnable(GL_SCISSOR_TEST);
        }

        loopi(2)
        {
            if(i && blurysize != blursize) setblurshader(i, texh, blurysize, bluryweights, bluryoffsets);
            else setblurshader(i, i ? texh : texw, blursize, blurweights, bluroffsets);

            if(!swaptexs() || rtsharefb) glFramebufferTexture2D_(GL_FRAMEBUFFER, GL_COLOR_ATTACHMENT0, GL_TEXTURE_2D, i ? rendertex : blurtex, 0);
            else glBindFramebuffer_(GL_FRAMEBUFFER, i ? renderfb : blurfb);
            glBindTexture(GL_TEXTURE_2D, i ? blurtex : rendertex);

            rendertiles();
        }

        if(scissor) glDisable(GL_SCISSOR_TEST);
            
        glEnable(GL_DEPTH_TEST);
        glEnable(GL_CULL_FACE);
    }

    virtual bool swaptexs() const { return false; }

    virtual bool dorender() { return true; }

    virtual bool shouldrender() { return true; }

    virtual void doblur(int blursize, float blursigma, int blurysize)
    { 
        int sx, sy, sw, sh;
        bool scissoring = rtscissor && scissorblur(sx, sy, sw, sh) && sw > 0 && sh > 0;
        if(!scissoring) { sx = sy = 0; sw = vieww; sh = viewh; }
        blur(blursize, blursigma, blurysize, sx, sy, sw, sh, scissoring);
    }

    virtual bool scissorrender(int &x, int &y, int &w, int &h)
    {
        if(scissorx1 >= scissorx2 || scissory1 >= scissory2) 
        {
            if(vieww < texw || viewh < texh)
            {
                x = y = 0;
                w = vieww;
                h = viewh;
                return true;
            }
            return false;
        }
        x = max(int(floor((scissorx1+1)/2*vieww)) - 2*blursize, 0);
        y = max(int(floor((scissory1+1)/2*viewh)) - 2*blursize, 0);
        w = min(int(ceil((scissorx2+1)/2*vieww)) + 2*blursize, vieww) - x;
        h = min(int(ceil((scissory2+1)/2*viewh)) + 2*blursize, viewh) - y;
        return true;
    }

    virtual bool scissorblur(int &x, int &y, int &w, int &h)
    {
        if(scissorx1 >= scissorx2 || scissory1 >= scissory2)
        {
            if(vieww < texw || viewh < texh)
            {
                x = y = 0;
                w = vieww;
                h = viewh;
                return true;
            }
            return false;
        }
        x = max(int(floor((scissorx1+1)/2*vieww)), 0);
        y = max(int(floor((scissory1+1)/2*viewh)), 0);
        w = min(int(ceil((scissorx2+1)/2*vieww)), vieww) - x;
        h = min(int(ceil((scissory2+1)/2*viewh)), viewh) - y;
        return true;
    }

    virtual void doclear() {}

    virtual bool screenrect() const { return false; }
    virtual bool filter() const { return true; }

    void render(int w, int h, int blursize = 0, float blursigma = 0, int blurysize = 0)
    {
        w = min(w, hwtexsize);
        h = min(h, hwtexsize);
        if(screenrect())
        {
            if(w > screenw) w = screenw;
            if(h > screenh) h = screenh;
        }
        vieww = w;
        viewh = h;
        if(w!=texw || h!=texh || (swaptexs() && !rtsharefb ? !blurfb : blurfb)) cleanup();
        
        if(!filter())
        {
            if(blurtex) cleanupblur();
            blursize = blurysize = 0;
        }
            
        if(!rendertex) setup(w, h);
   
        scissorx2 = scissory2 = -1;
        scissorx1 = scissory1 = 1;
        memset(blurtiles, 0, sizeof(blurtiles));
 
        if(!shouldrender()) return;

        if(blursize && !blurtex) setupblur();
        if(swaptexs() && blursize)
        {
            swap(rendertex, blurtex);
            if(!rtsharefb)
            {
                swap(renderfb, blurfb);
                swap(renderdb, blurdb);
            }
        }
        glBindFramebuffer_(GL_FRAMEBUFFER, renderfb);
        if(swaptexs() && blursize && rtsharefb)
            glFramebufferTexture2D_(GL_FRAMEBUFFER, attachment(), GL_TEXTURE_2D, rendertex, 0);
        glViewport(0, 0, vieww, viewh);

        doclear();

        int sx, sy, sw, sh;
        bool scissoring = rtscissor && scissorrender(sx, sy, sw, sh) && sw > 0 && sh > 0;
        if(scissoring)
        {
            glScissor(sx, sy, sw, sh);
            glEnable(GL_SCISSOR_TEST);
        }
        else
        {
            sx = sy = 0;
            sw = vieww;
            sh = viewh;
        }

        if(!depthtest()) glDisable(GL_DEPTH_TEST);

        bool succeeded = dorender();

        if(!depthtest()) glEnable(GL_DEPTH_TEST);

        if(scissoring) glDisable(GL_SCISSOR_TEST);

        if(succeeded)
        {
            initialized = true;

            if(blursize) doblur(blursize, blursigma, blurysize ? blurysize : blursize);
        }

        glBindFramebuffer_(GL_FRAMEBUFFER, 0);
        glViewport(0, 0, screenw, screenh);
    }

    virtual void dodebug(int w, int h) {}
    virtual bool flipdebug() const { return true; }

    void debugscissor(int w, int h, bool lines = false)
    {
        if(!rtscissor || scissorx1 >= scissorx2 || scissory1 >= scissory2) return;
        int sx = int(0.5f*(scissorx1 + 1)*w),
            sy = int(0.5f*(scissory1 + 1)*h),
            sw = int(0.5f*(scissorx2 - scissorx1)*w),
            sh = int(0.5f*(scissory2 - scissory1)*h);
        if(flipdebug()) { sy = h - sy; sh = -sh; }
        gle::defvertex(2);
        gle::begin(lines ? GL_LINE_LOOP : GL_TRIANGLE_STRIP);
        gle::attribf(sx,      sy);
        gle::attribf(sx + sw, sy);
        if(lines) gle::attribf(sx + sw, sy + sh);
        gle::attribf(sx,      sy + sh);
        if(!lines) gle::attribf(sx + sw, sy + sh);
        gle::end();
    }

    void debugblurtiles(int w, int h, bool lines = false)
    {
        if(!blurtile) return;
        float vxsz = float(w)/BLURTILES, vysz = float(h)/BLURTILES;
        gle::defvertex(2);
        loop(y, BLURTILES+1)
        {
            uint mask = blurtiles[y];
            int x = 0;
            while(mask)
            {
                while(!(mask&0xFF)) { mask >>= 8; x += 8; }
                while(!(mask&1)) { mask >>= 1; x++; }
                    int xstart = x;
                do { mask >>= 1; x++; } while(mask&1);
                uint strip = (BLURTILEMASK>>(BLURTILES - x)) & (BLURTILEMASK<<xstart);
                int yend = y;
                do { blurtiles[yend] &= ~strip; yend++; } while((blurtiles[yend] & strip) == strip);
                float vx = xstart*vxsz,
                      vy = y*vysz,
                      vw = (x-xstart)*vxsz,
                      vh = (yend-y)*vysz;
                if(flipdebug()) { vy = h - vy; vh = -vh; }
                loopi(lines ? 1 : 2)
                {
                    if(!lines) gle::colorf(1, 1, i ? 1.0f : 0.5f);
                    gle::begin(lines || i ? GL_LINE_LOOP : GL_TRIANGLE_STRIP);
                    gle::attribf(vx,    vy);
                    gle::attribf(vx+vw, vy);
                    if(lines || i) gle::attribf(vx+vw, vy+vh);
                    gle::attribf(vx,    vy+vh);
                    if(!lines && !i) gle::attribf(vx+vw, vy+vh);
                    gle::end();
                }
            }
        }
    }

    void debug()
    {
        if(!rendertex) return;
        int w = min(screenw, screenh)/2, h = (w*screenh)/screenw;
        hudshader->set(); 
        gle::colorf(1, 1, 1);
        glBindTexture(GL_TEXTURE_2D, rendertex);
        float tx1 = 0, tx2 = 1, ty1 = 0, ty2 = 1;
        if(flipdebug()) swap(ty1, ty2);
        hudquad(0, 0, w, h, tx1, ty1, tx2-tx1, ty2-ty1);
        hudnotextureshader->set();
        dodebug(w, h);
    }
};

