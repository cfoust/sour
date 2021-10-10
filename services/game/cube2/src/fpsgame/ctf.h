#ifndef PARSEMESSAGES

#define ctfteamflag(s) (!strcmp(s, "good") ? 1 : (!strcmp(s, "evil") ? 2 : 0))
#define ctfflagteam(i) (i==1 ? "good" : (i==2 ? "evil" : NULL))

#ifdef SERVMODE
VAR(ctftkpenalty, 0, 1, 1);

struct ctfservmode : servmode
#else
struct ctfclientmode : clientmode
#endif
{
    static const int BASERADIUS = 64;
    static const int BASEHEIGHT = 24;
    static const int MAXFLAGS = 20;
    static const int FLAGRADIUS = 16;
    static const int FLAGLIMIT = 10;
    static const int MAXHOLDSPAWNS = 100;
    static const int HOLDSECS = 20;
    static const int HOLDFLAGS = 1;
    static const int HOLDDEATHPENALTY = 5;
    static const int RESPAWNSECS = 5;

    struct flag
    {
        int id, version, spawnindex;
        vec droploc, spawnloc;
        int team, droptime, owntime, holdtime;
#ifdef SERVMODE
        int owner, dropcount, dropper, invistime;
#else
        fpsent *owner;
        float dropangle, spawnangle;
        entitylight light;
        vec interploc;
        float interpangle;
        int interptime, vistime;
#endif

        flag() : id(-1) { reset(); }

        void reset()
        {
            version = 0;
            spawnindex = -1;
            droploc = spawnloc = vec(0, 0, 0);
#ifdef SERVMODE
            dropcount = 0;
            owner = dropper = -1;
            invistime = 0;
#else
            if(id >= 0) loopv(players) players[i]->flagpickup &= ~(1<<id);
            owner = NULL;
            dropangle = spawnangle = 0;
            interploc = vec(0, 0, 0);
            interpangle = 0;
            interptime = 0;
            vistime = -1000;
#endif
            team = 0;
            droptime = owntime = holdtime = 0;
        }

#ifndef SERVMODE
        vec pos() const
        {
        	if(owner) return vec(owner->o).sub(owner->eyeheight);
        	if(droptime) return droploc;
        	return spawnloc;
        }
#endif

        bool hasowner() const
        {
#ifdef SERVMODE
            return owner>=0;
#else
            return !!owner;
#endif
        }

        int holdcounter() const
        {
            if(!holdtime) return 0;
            if(!hasowner() && droptime && droptime >= holdtime) return max((droptime - holdtime) - (lastmillis - droptime), 0);
            return lastmillis - holdtime;
        }
    };

    struct holdspawn
    {
        vec o;
#ifndef SERVMODE
        entitylight light;
#endif
    };
    vector<holdspawn> holdspawns;
    vector<flag> flags;
    int scores[2];

    void resetflags()
    {
        holdspawns.shrink(0);
        flags.shrink(0);
        loopk(2) scores[k] = 0;
    }

#ifdef SERVMODE
    bool addflag(int i, const vec &o, int team, int invistime = 0)
#else
    bool addflag(int i, const vec &o, int team, int vistime = -1000)
#endif
    {
        if(i<0 || i>=MAXFLAGS) return false;
        while(flags.length()<=i) flags.add();
        flag &f = flags[i];
        f.id = i;
        f.reset();
        f.team = team;
        f.spawnloc = o;
#ifdef SERVMODE
        f.invistime = invistime;
#else
        f.vistime = vistime;
#endif
        return true;
    }

#ifdef SERVMODE
    bool addholdspawn(const vec &o)
#else
    bool addholdspawn(const vec &o)
#endif
    {
        if(holdspawns.length() >= MAXHOLDSPAWNS) return false;
        holdspawn &h = holdspawns.add();
        h.o = o;
        return true;
    }

#ifdef SERVMODE
    void ownflag(int i, int owner, int owntime, int ownteam = -1)
#else
    void ownflag(int i, fpsent *owner, int owntime, int ownteam = -1)
#endif
    {
        flag &f = flags[i];
        f.owner = owner;
        f.owntime = owntime;
        if(f.holdtime && f.team == ownteam)
        {
            if(f.droptime && f.droptime < owntime) f.holdtime += 2*(owntime - f.droptime);
            f.holdtime = min(f.holdtime, owntime);
        }
        else
        {
            f.holdtime = owntime;
            if(ownteam >= 0) f.team = ownteam;
        }
#ifdef SERVMODE
        if(owner == f.dropper) { if(f.dropcount < INT_MAX) f.dropcount++; }
        else f.dropcount = 0;
        f.dropper = -1;
        f.invistime = 0;
#else
        loopv(players) players[i]->flagpickup &= ~(1<<f.id);
        if(!f.vistime) f.vistime = owntime;
#endif
    }

#ifdef SERVMODE
    void dropflag(int i, const vec &o, int droptime, int dropper = -1, bool penalty = false)
#else
    void dropflag(int i, const vec &o, float yaw, int droptime)
#endif
    {
        flag &f = flags[i];
        f.droploc = o;
        f.droptime = droptime;
#ifdef SERVMODE
        if(dropper < 0) f.dropcount = 0;
        else if(penalty) f.dropcount = INT_MAX;
        f.dropper = dropper;
        f.owner = -1;
        f.invistime = 0;
#else
        loopv(players) players[i]->flagpickup &= ~(1<<f.id);
        f.owner = NULL;
        f.dropangle = yaw;
        if(!f.vistime) f.vistime = droptime;
#endif
    }

#ifdef SERVMODE
    void returnflag(int i, int invistime = 0)
#else
    void returnflag(int i, int vistime = -1000)
#endif
    {
        flag &f = flags[i];
        f.droptime = 0;
        f.holdtime = 0;
#ifdef SERVMODE
        f.dropcount = 0;
        f.owner = f.dropper = -1;
        f.invistime = invistime;
#else
        loopv(players) players[i]->flagpickup &= ~(1<<f.id);
        f.vistime = vistime;
        f.owner = NULL;
#endif
    }

    void deadflag(int i)
    {
        flag &f = flags[i];
        if(f.holdtime)
        {
            int limit = !f.hasowner() && f.droptime && f.droptime >= f.holdtime ? f.droptime : lastmillis;
            f.holdtime += HOLDDEATHPENALTY*1000;
            if(f.holdtime >= limit) f.holdtime = 0;
        }
    }

    int totalscore(int team)
    {
        return team >= 1 && team <= 2 ? scores[team-1] : 0;
    }

    int setscore(int team, int score)
    {
        if(team >= 1 && team <= 2) return scores[team-1] = score;
        return 0;
    }

    int addscore(int team, int score)
    {
        if(team >= 1 && team <= 2) return scores[team-1] += score;
        return 0;
    }

    bool hidefrags() { return true; }

    int getteamscore(const char *team)
    {
        return totalscore(ctfteamflag(team));
    }

    void getteamscores(vector<teamscore> &tscores)
    {
        loopk(2) if(scores[k]) tscores.add(teamscore(ctfflagteam(k+1), scores[k]));
    }

    bool insidebase(const flag &f, const vec &o)
    {
        float dx = (f.spawnloc.x-o.x), dy = (f.spawnloc.y-o.y), dz = (f.spawnloc.z-o.z);
        return dx*dx + dy*dy <= BASERADIUS*BASERADIUS && fabs(dz) <= BASEHEIGHT;
    }

#ifdef SERVMODE
    static const int RESETFLAGTIME = 10000;
    static const int INVISFLAGTIME = 20000;

    bool notgotflags;

    ctfservmode() : notgotflags(false) {}

    void reset(bool empty)
    {
        resetflags();
        notgotflags = !empty;
    }

    void cleanup()
    {
        reset(false);
    }

    void setupholdspawns()
    {
        if(!m_hold || holdspawns.empty()) return;
        while(flags.length() < HOLDFLAGS)
        {
            int i = flags.length();
            if(!addflag(i, vec(0, 0, 0), 0, 0)) break;
            flag &f = flags[i];
            spawnflag(i);
            sendf(-1, 1, "ri6", N_RESETFLAG, i, ++f.version, f.spawnindex, 0, 0);
        }
    }

    void setup()
    {
        reset(false);
        if(notgotitems || ments.empty()) return;
        if(m_hold)
        {
            loopv(ments)
            {
                entity &e = ments[i];
                if(e.type != BASE) continue;
                if(!addholdspawn(e.o)) break;
            }
            setupholdspawns();
        }
        else loopv(ments)
        {
            entity &e = ments[i];
            if(e.type != FLAG || e.attr2 < 1 || e.attr2 > 2) continue;
            if(!addflag(flags.length(), e.o, e.attr2, m_protect ? lastmillis : 0)) break;
        }
        notgotflags = false;
    }

    void newmap()
    {
        reset(true);
    }

    void dropflag(clientinfo *ci, clientinfo *dropper = NULL)
    {
        if(notgotflags) return;
        loopv(flags) if(flags[i].owner==ci->clientnum)
        {
            flag &f = flags[i];
            if(m_protect && insidebase(f, ci->state.o))
            {
                returnflag(i);
                sendf(-1, 1, "ri4", N_RETURNFLAG, ci->clientnum, i, ++f.version);
            }
            else
            {
                ivec o(vec(ci->state.o).mul(DMF));
                sendf(-1, 1, "ri7", N_DROPFLAG, ci->clientnum, i, ++f.version, o.x, o.y, o.z);
                dropflag(i, vec(o).div(DMF), lastmillis, dropper ? dropper->clientnum : ci->clientnum, dropper && dropper!=ci);
            }
        }
    }

    void leavegame(clientinfo *ci, bool disconnecting = false)
    {
        dropflag(ci);
        loopv(flags) if(flags[i].dropper == ci->clientnum) { flags[i].dropper = -1; flags[i].dropcount = 0; }
    }

    void died(clientinfo *ci, clientinfo *actor)
    {
        if(notgotflags) return;
        if(m_hold) loopv(flags) if(flags[i].owner==ci->clientnum) deadflag(i);
        dropflag(ci, ctftkpenalty && actor && actor != ci && isteam(actor->team, ci->team) ? actor : NULL);
        loopv(flags) if(flags[i].dropper == ci->clientnum) { flags[i].dropper = -1; flags[i].dropcount = 0; }
    }

    bool canspawn(clientinfo *ci, bool connecting) 
    { 
        return m_efficiency || !m_protect ? connecting || !ci->state.lastdeath || gamemillis+curtime-ci->state.lastdeath >= RESPAWNSECS*1000 : true;
    }

    bool canchangeteam(clientinfo *ci, const char *oldteam, const char *newteam)
    {
        return ctfteamflag(newteam) > 0;
    }

    void changeteam(clientinfo *ci, const char *oldteam, const char *newteam)
    {
        dropflag(ci);
    }

    void spawnflag(int i)
    {
        if(holdspawns.empty()) return;
        int spawnindex = flags[i].spawnindex;
        loopj(4)
        {
            spawnindex = rnd(holdspawns.length());
            if(spawnindex != flags[i].spawnindex) break;
        }
        flags[i].spawnindex = spawnindex;
    }

    void scoreflag(clientinfo *ci, int goal, int relay = -1)
    {
        returnflag(relay >= 0 ? relay : goal, m_protect ? lastmillis : 0);
        ci->state.flags++;
        int team = ctfteamflag(ci->team), score = addscore(team, 1);
        if(m_hold) spawnflag(goal);
        sendf(-1, 1, "rii9", N_SCOREFLAG, ci->clientnum, relay, relay >= 0 ? ++flags[relay].version : -1, goal, ++flags[goal].version, flags[goal].spawnindex, team, score, ci->state.flags);
        if(score >= FLAGLIMIT) startintermission();
    }

    void takeflag(clientinfo *ci, int i, int version)
    {
        if(notgotflags || !flags.inrange(i) || ci->state.state!=CS_ALIVE || !ci->team[0]) return;
        flag &f = flags[i];
        if((m_hold ? f.spawnindex < 0 : !ctfflagteam(f.team)) || f.owner>=0 || f.version != version || (f.droptime && f.dropper == ci->clientnum && f.dropcount >= 3)) return;
        int team = ctfteamflag(ci->team);
        if(m_hold || m_protect == (f.team==team))
        {
            loopvj(flags) if(flags[j].owner==ci->clientnum) return;
            ownflag(i, ci->clientnum, lastmillis, m_hold ? team : -1);
            sendf(-1, 1, "ri4", N_TAKEFLAG, ci->clientnum, i, ++f.version);
        }
        else if(m_protect)
        {
            if(!f.invistime) scoreflag(ci, i);
        }
        else if(f.droptime)
        {
            returnflag(i);
            sendf(-1, 1, "ri4", N_RETURNFLAG, ci->clientnum, i, ++f.version);
        }
        else
        {
            loopvj(flags) if(flags[j].owner==ci->clientnum) { scoreflag(ci, i, j); break; }
        }
    }

    void update()
    {
        if(gamemillis>=gamelimit || notgotflags) return;
        loopv(flags)
        {
            flag &f = flags[i];
            if(f.owner<0 && f.droptime && lastmillis - f.droptime >= RESETFLAGTIME)
            {
                returnflag(i, m_protect ? lastmillis : 0);
                if(m_hold) spawnflag(i);
                sendf(-1, 1, "ri6", N_RESETFLAG, i, ++f.version, f.spawnindex, m_hold ? 0 : f.team, m_hold ? 0 : addscore(f.team, m_protect ? -1 : 0));
            }
            if(f.invistime && lastmillis - f.invistime >= INVISFLAGTIME)
            {
                f.invistime = 0;
                sendf(-1, 1, "ri3", N_INVISFLAG, i, 0);
            }
            if(m_hold && f.owner>=0 && lastmillis - f.holdtime >= HOLDSECS*1000)
            {
                clientinfo *ci = getinfo(f.owner);
                if(ci) scoreflag(ci, i);
                else
                {
                    spawnflag(i);
                    sendf(-1, 1, "ri6", N_RESETFLAG, i, ++f.version, f.spawnindex, 0, 0);
                }
            }
        }
    }

    void initclient(clientinfo *ci, packetbuf &p, bool connecting)
    {
        putint(p, N_INITFLAGS);
        loopk(2) putint(p, scores[k]);
        putint(p, flags.length());
        loopv(flags)
        {
            flag &f = flags[i];
            putint(p, f.version);
            putint(p, f.spawnindex);
            putint(p, f.owner);
            putint(p, f.invistime ? 1 : 0);
            if(f.owner<0)
            {
                putint(p, f.droptime ? 1 : 0);
                if(f.droptime)
                {
                    putint(p, int(f.droploc.x*DMF));
                    putint(p, int(f.droploc.y*DMF));
                    putint(p, int(f.droploc.z*DMF));
                }
            }
            if(m_hold)
            {
                if(!f.team || !f.holdtime) putint(p, -1);
                else
                {
                    putint(p, f.team);
                    putint(p, f.holdcounter() / 100);
                }
            }
        }
    }

    void parseflags(ucharbuf &p, bool commit)
    {
        int numflags = getint(p);
        loopi(numflags)
        {
            int team = getint(p);
            vec o;
            loopk(3) o[k] = max(getint(p)/DMF, 0.0f);
            if(p.overread()) break;
            if(commit && notgotflags)
            {
                if(m_hold) addholdspawn(o);
                else addflag(i, o, team, m_protect ? lastmillis : 0);
            }
        }
        if(commit && notgotflags)
        {
            if(m_hold) setupholdspawns();
            notgotflags = false;
        }
    }
};
#else
    void preload()
    {
        if(m_hold) preloadmodel("flags/neutral");
        else
        {
            preloadmodel("flags/red");
            preloadmodel("flags/blue");
        }
        static const int sounds[] = { S_FLAGPICKUP, S_FLAGDROP, S_FLAGRETURN, S_FLAGSCORE, S_FLAGRESET, S_FLAGFAIL };
        loopi(sizeof(sounds)/sizeof(sounds[0])) preloadsound(sounds[i]);
    }

    void drawblip(fpsent *d, float x, float y, float s, const vec &pos, bool flagblip)
    {
        float scale = calcradarscale();
        vec dir = d->o;
        dir.sub(pos).div(scale);
        float size = flagblip ? 0.1f : 0.05f,
              xoffset = flagblip ? -2*(3/32.0f)*size : -size,
              yoffset = flagblip ? -2*(1 - 3/32.0f)*size : -size,
              dist = dir.magnitude2(), maxdist = 1 - 0.05f - 0.05f;
        if(dist >= maxdist) dir.mul(maxdist/dist);
        dir.rotate_around_z(-camera1->yaw*RAD);
        drawradar(x + s*0.5f*(1.0f + dir.x + xoffset), y + s*0.5f*(1.0f + dir.y + yoffset), size*s);
    }

    void drawblip(fpsent *d, float x, float y, float s, int i, bool flagblip)
    {
        flag &f = flags[i];
        settexture(m_hold && (!flagblip || !f.owner || lastmillis%1000 < 500) ? (flagblip ? "packages/hud/blip_neutral_flag.png" : "packages/hud/blip_neutral.png") :
                    ((m_hold ? ctfteamflag(f.owner->team) : f.team)==ctfteamflag(player1->team) ?
                        (flagblip ? "packages/hud/blip_blue_flag.png" : "packages/hud/blip_blue.png") :
                        (flagblip ? "packages/hud/blip_red_flag.png" : "packages/hud/blip_red.png")), 3);
        drawblip(d, x, y, s, flagblip ? (f.owner ? f.owner->o : (f.droptime ? f.droploc : f.spawnloc)) : f.spawnloc, flagblip);
    }

    int clipconsole(int w, int h)
    {
        return (h*(1 + 1 + 10))/(4*10);
    }

    void drawhud(fpsent *d, int w, int h)
    {
        if(d->state == CS_ALIVE)
        {
            loopv(flags) if(flags[i].owner == d)
            {
                flag &f = flags[i];
                int x = HICON_X + 3*HICON_STEP + (d->quadmillis ? HICON_SIZE + HICON_SPACE : 0);
                drawicon(m_hold ? HICON_NEUTRAL_FLAG : (f.team==ctfteamflag(player1->team) ? HICON_BLUE_FLAG : HICON_RED_FLAG), x, HICON_Y);
                if(m_hold)
                {
                    pushhudmatrix();
                    hudmatrix.scale(2, 2, 1);
                    flushhudmatrix();
                    draw_textf("%d", (x + HICON_SIZE + HICON_SPACE)/2, HICON_TEXTY/2, max(HOLDSECS - f.holdcounter()/1000, 0));
                    pophudmatrix();
                }
                break;
            }
        }

        glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);
        int s = 1800/4, x = 1800*w/h - s - s/10, y = s/10;
        gle::colorf(1, 1, 1, minimapalpha);
        if(minimapalpha >= 1) glDisable(GL_BLEND);
        bindminimap();
        drawminimap(d, x, y, s);
        if(minimapalpha >= 1) glEnable(GL_BLEND);
        gle::colorf(1, 1, 1);
        float margin = 0.04f, roffset = s*margin, rsize = s + 2*roffset;
        settexture("packages/hud/radar.png", 3);
        drawradar(x - roffset, y - roffset, rsize);
        #if 0
        settexture("packages/hud/compass.png", 3);
        pushhudmatrix();
        hudmatrix.translate(x - roffset + 0.5f*rsize, y - roffset + 0.5f*rsize, 0);
        hudmatrix.rotate_around_z((camera1->yaw + 180)*-RAD);
        flushhudmatrix();
        drawradar(-0.5f*rsize, -0.5f*rsize, rsize);
        pophudmatrix();
        #endif
        if(m_hold)
        {
            settexture("packages/hud/blip_neutral.png", 3);
            loopv(holdspawns) drawblip(d, x, y, s, holdspawns[i].o, false);
        }
        loopv(flags)
        {
            flag &f = flags[i];
            if(m_hold ? f.spawnindex < 0 : !ctfflagteam(f.team)) continue;
            if(!m_hold) drawblip(d, x, y, s, i, false);
            if(f.owner)
            {
                if(!m_hold && lastmillis%1000 >= 500) continue;
            }
            else if(f.droptime && (f.droploc.x < 0 || lastmillis%300 >= 150)) continue;
            drawblip(d, x, y, s, i, true);
        }
        drawteammates(d, x, y, s);
        if(d->state == CS_DEAD && (m_efficiency || !m_protect))
        {
            int wait = respawnwait(d);
            if(wait>=0)
            {
                pushhudmatrix();
                hudmatrix.scale(2, 2, 1);
                flushhudmatrix();
                bool flash = wait>0 && d==player1 && lastspawnattempt>=d->lastpain && lastmillis < lastspawnattempt+100;
                draw_textf("%s%d", (x+s/2)/2-(wait>=10 ? 28 : 16), (y+s/2)/2-32, flash ? "\f3" : "", wait);
                pophudmatrix();
            }
        }
    }

    void removeplayer(fpsent *d)
    {
        loopv(flags) if(flags[i].owner == d)
        {
            flag &f = flags[i];
            f.interploc.x = -1;
            f.interptime = 0;
            dropflag(i, f.owner->o, f.owner->yaw, 1);
        }
    }

    void died(fpsent *victim, fpsent *actor)
    {
        if(m_hold) loopv(flags) if(flags[i].owner == victim) deadflag(i);
    }

    vec interpflagpos(flag &f, float &angle)
    {
        vec pos = f.owner ? vec(f.owner->abovehead()).add(vec(0, 0, 1)) : (f.droptime ? f.droploc : f.spawnloc);
        if(f.owner) angle = f.owner->yaw;
        else if(m_hold)
        {
            float yaw, pitch;
            vectoyawpitch(vec(pos).sub(camera1->o), yaw, pitch);
            angle = yaw + 180;
        }
        else angle = f.droptime ? f.dropangle : f.spawnangle;
        if(pos.x < 0) return pos;
        if(f.interptime && f.interploc.x >= 0)
        {
            float t = min((lastmillis - f.interptime)/500.0f, 1.0f);
            pos.lerp(f.interploc, pos, t);
            angle += (1-t)*(f.interpangle - angle);
        }
        return pos;
    }

    vec interpflagpos(flag &f) { float angle; return interpflagpos(f, angle); }

    void rendergame()
    {
        loopv(flags)
        {
            flag &f = flags[i];
            if(!f.owner && f.droptime && f.droploc.x < 0) continue;
            if(m_hold && f.spawnindex < 0) continue;
            const char *flagname = m_hold && (!f.owner || lastmillis%1000 < 500) ? "flags/neutral" : (m_hold ? ctfteamflag(f.owner->team) : f.team)==ctfteamflag(player1->team) ? "flags/blue" : "flags/red";
            float angle;
            vec pos = interpflagpos(f, angle);
            if(m_hold)
                rendermodel(!f.droptime && !f.owner ? &f.light : NULL, flagname, ANIM_MAPMODEL|ANIM_LOOP,
                        pos, angle, 0,
                        MDL_GHOST | MDL_CULL_VFC | (f.droptime || f.owner ? MDL_LIGHT : 0),
                        NULL, NULL, 0, 0, 0.5f + 0.5f*(2*fabs(fmod(lastmillis/1000.0f, 1.0f) - 0.5f)));
            rendermodel(!f.droptime && !f.owner ? &f.light : NULL, flagname, ANIM_MAPMODEL|ANIM_LOOP,
                        pos, angle, 0,
                        MDL_DYNSHADOW | MDL_CULL_VFC | MDL_CULL_OCCLUDED | (f.droptime || f.owner ? MDL_LIGHT : 0),
                        NULL, NULL, 0, 0, 0.3f + (f.vistime ? 0.7f*min((lastmillis - f.vistime)/1000.0f, 1.0f) : 0.0f));


            if(m_hold && canaddparticles() && f.team && f.holdtime)
            {
                int counter = f.holdcounter();
                if(counter > 0)
                {
                    vec above = vec(pos).addz(3.0f);
                    abovemodel(above, flagname);
                    defformatstring(msg, "%d", max(HOLDSECS - counter / 1000, 0));
                    particle_textcopy(above, msg, PART_TEXT, 1, f.team == ctfteamflag(player1->team) ? 0x6496FF : 0xFF4B19, 4.0f);
                }
            }
            if(m_protect && canaddparticles() && f.owner && insidebase(f, f.owner->feetpos()))
            {
                particle_flare(pos, f.spawnloc, 0, PART_LIGHTNING, strcmp(f.owner->team, player1->team) ? 0xFF2222 : 0x2222FF, 1.0f);
                if(!flags.inrange(f.owner->lastbase))
                {
                    particle_fireball(pos, 4.8f, PART_EXPLOSION, 250, strcmp(f.owner->team, player1->team) ? 0x802020 : 0x2020FF, 4.8f);
                    particle_splash(PART_SPARK, 50, 250, pos, strcmp(f.owner->team, player1->team) ? 0x802020 : 0x2020FF, 0.24f);
                }
                f.owner->lastbase = i;
            }
        }
        if(m_protect && canaddparticles()) loopv(players)
        {
            fpsent *d = players[i];
            if(!flags.inrange(d->lastbase)) continue;
            flag &f = flags[d->lastbase];
            if(f.owner == d && insidebase(f, d->feetpos())) continue;
            d->lastbase = -1;
            float angle;
            vec pos = interpflagpos(f, angle);
            particle_fireball(pos, 4.8f, PART_EXPLOSION, 250, strcmp(d->team, player1->team) ? 0x802020 : 0x2020FF, 4.8f);
            particle_splash(PART_SPARK, 50, 250, pos, strcmp(d->team, player1->team) ? 0x802020 : 0x2020FF, 0.24f);
        }
    }

    void setup()
    {
        resetflags();
        if(m_hold)
        {
            loopv(entities::ents)
            {
                extentity *e = entities::ents[i];
                if(e->type!=BASE) continue;
                if(!addholdspawn(e->o)) continue;
                holdspawns.last().light = e->light;
            }
            if(holdspawns.length()) while(flags.length() < HOLDFLAGS) addflag(flags.length(), vec(0, 0, 0), 0, -1000);
        }
        else
        {
            loopv(entities::ents)
            {
                extentity *e = entities::ents[i];
                if(e->type!=FLAG) continue;
                if(e->attr2<1 || e->attr2>2) continue;
                int index = flags.length();
                if(!addflag(index, e->o, e->attr2, m_protect ? 0 : -1000)) continue;
                flags[index].spawnangle = e->attr1;
                flags[index].light = e->light;
            }
        }
    }

    void senditems(packetbuf &p)
    {
        putint(p, N_INITFLAGS);
        if(m_hold)
        {
            putint(p, holdspawns.length());
            loopv(holdspawns)
            {
                holdspawn &h = holdspawns[i];
                putint(p, -1);
                loopk(3) putint(p, int(h.o[k]*DMF));
            }
        }
        else
        {
            putint(p, flags.length());
            loopv(flags)
            {
                flag &f = flags[i];
                putint(p, f.team);
                loopk(3) putint(p, int(f.spawnloc[k]*DMF));
            }
        }
    }

    void parseflags(ucharbuf &p, bool commit)
    {
        loopk(2)
        {
            int score = getint(p);
            if(commit) scores[k] = score;
        }
        int numflags = getint(p);
        loopi(numflags)
        {
            int version = getint(p), spawn = getint(p), owner = getint(p), invis = getint(p), dropped = 0, holdteam = -1, holdtime = 0;
            vec droploc(0, 0, 0);
            if(owner<0)
            {
                dropped = getint(p);
                if(dropped) loopk(3) droploc[k] = getint(p)/DMF;
            }
            if(m_hold)
            {
                holdteam = getint(p);
                if(holdteam >= 0) holdtime = getint(p)*100;
            }
            if(p.overread()) break;
            if(commit && flags.inrange(i))
            {
                flag &f = flags[i];
                f.version = version;
                f.spawnindex = spawn;
                if(m_hold) spawnflag(f);
                f.owner = owner>=0 ? (owner==player1->clientnum ? player1 : newclient(owner)) : NULL;
                f.owntime = owner>=0 ? lastmillis - (m_hold ? holdtime : 0) : 0;
                if(m_hold)
                {
                    if(holdteam >= 0) { f.team = holdteam; f.holdtime = lastmillis - holdtime; }
                    else { f.team = 0; f.holdtime = 0; }
                }
                f.droptime = dropped ? lastmillis : 0;
                f.droploc = dropped ? droploc : f.spawnloc;
                f.vistime = invis>0 ? 0 : -1000;
                f.interptime = 0;

                if(dropped)
                {
                    f.droploc.z += 4;
                    if(!droptofloor(f.droploc, 4, 0)) f.droploc = vec(-1, -1, -1);
                }
            }
        }
    }

    void trydropflag()
    {
        if(!m_ctf) return;
        loopv(flags) if(flags[i].owner == player1)
        {
            addmsg(N_TRYDROPFLAG, "rc", player1);
            return;
        }
    }

    const char *teamcolorflag(flag &f)
    {
        return m_hold ? "the flag" : teamcolor("your flag", ctfflagteam(f.team), "the enemy flag");
    }

    void dropflag(fpsent *d, int i, int version, const vec &droploc)
    {
        if(!flags.inrange(i)) return;
        flag &f = flags[i];
        f.version = version;
        f.interploc = interpflagpos(f, f.interpangle);
        f.interptime = lastmillis;
        dropflag(i, droploc, d->yaw, lastmillis);
        f.droploc.z += 4;
        d->flagpickup |= 1<<f.id;
        if(!droptofloor(f.droploc, 4, 0))
        {
            f.droploc = vec(-1, -1, -1);
            f.interptime = 0;
        }
        conoutf(CON_GAMEINFO, "%s dropped %s", teamcolorname(d), teamcolorflag(f));
        teamsound(d, S_FLAGDROP);
    }

    void flagexplosion(int i, int team, const vec &loc)
    {
        int fcolor;
        vec color;
        if(!team) { fcolor = 0xFF8080; color = vec(1, 0.75f, 0.5f); }
        else if(team==ctfteamflag(player1->team)) { fcolor = 0x2020FF; color = vec(0.25f, 0.25f, 1); }
        else { fcolor = 0x802020; color = vec(1, 0.25f, 0.25f); }
        particle_fireball(loc, 30, PART_EXPLOSION, -1, fcolor, 4.8f);
        adddynlight(loc, 35, color, 900, 100);
        particle_splash(PART_SPARK, 150, 300, loc, fcolor, 0.24f);
    }

    void flageffect(int i, int team, const vec &from, const vec &to)
    {
        vec fromexp(from), toexp(to);
        if(from.x >= 0)
        {
            fromexp.z += 8;
            flagexplosion(i, team, fromexp);
        }
        if(from==to) return;
        if(to.x >= 0)
        {
            toexp.z += 8;
            flagexplosion(i, team, toexp);
        }
        if(from.x >= 0 && to.x >= 0)
            particle_flare(fromexp, toexp, 600, PART_LIGHTNING, !team ? 0xFFC0A0 : (team==ctfteamflag(player1->team) ? 0x2222FF : 0xFF2222), 1.0f);
    }

    void returnflag(fpsent *d, int i, int version)
    {
        if(!flags.inrange(i)) return;
        flag &f = flags[i];
        f.version = version;
        flageffect(i, f.team, interpflagpos(f), f.spawnloc);
        f.interptime = 0;
        returnflag(i);
        if(m_protect && d->feetpos().dist(f.spawnloc) < FLAGRADIUS) d->flagpickup |= 1<<f.id;
        conoutf(CON_GAMEINFO, "%s returned %s", teamcolorname(d), teamcolorflag(f));
        teamsound(d, S_FLAGRETURN);
    }

    void spawnflag(flag &f)
    {
        if(!holdspawns.inrange(f.spawnindex)) return;
        holdspawn &h = holdspawns[f.spawnindex];
        f.spawnloc = h.o;
        f.light = h.light;
    }

    void resetflag(int i, int version, int spawnindex, int team, int score)
    {
        setscore(team, score);
        if(!flags.inrange(i)) return;
        flag &f = flags[i];
        f.version = version;
        bool shouldeffect = !m_hold || f.spawnindex >= 0;
        f.spawnindex = spawnindex;
        if(m_hold) spawnflag(f);
        if(shouldeffect) flageffect(i, m_hold ? 0 : team, interpflagpos(f), f.spawnloc);
        f.interptime = 0;
        returnflag(i, m_protect ? 0 : -1000);
        if(shouldeffect)
        {
            conoutf(CON_GAMEINFO, "%s reset", teamcolorflag(f));
            teamsound(team == ctfteamflag(player1->team), S_FLAGRESET);
        }
    }

    void scoreflag(fpsent *d, int relay, int relayversion, int goal, int goalversion, int goalspawn, int team, int score, int dflags)
    {
        setscore(team, score);
        if(flags.inrange(goal))
        {
            flag &f = flags[goal];
            f.version = goalversion;
            f.spawnindex = goalspawn;
            if(m_hold) spawnflag(f);
            if(relay >= 0)
            {
                flags[relay].version = relayversion;
                flageffect(goal, team, f.spawnloc, flags[relay].spawnloc);
            }
            else flageffect(goal, team, interpflagpos(f), f.spawnloc);
            f.interptime = 0;
            returnflag(relay >= 0 ? relay : goal, m_protect ? 0 : -1000);
            d->flagpickup &= ~(1<<f.id);
            if(d->feetpos().dist(f.spawnloc) < FLAGRADIUS) d->flagpickup |= 1<<f.id;
        }
        if(d!=player1)
        {
            defformatstring(ds, "%d", score);
            particle_textcopy(d->abovehead(), ds, PART_TEXT, 2000, 0x32FF64, 4.0f, -8);
        }
        d->flags = dflags;
        conoutf(CON_GAMEINFO, "%s scored for %s", teamcolorname(d), teamcolor("your team", ctfflagteam(team), "the enemy team"));
        playsound(team==ctfteamflag(player1->team) ? S_FLAGSCORE : S_FLAGFAIL);

        if(score >= FLAGLIMIT) conoutf(CON_GAMEINFO, "%s captured %d flags", teamcolor("your team", ctfflagteam(team), "the enemy team"), score);
    }

    void takeflag(fpsent *d, int i, int version)
    {
        if(!flags.inrange(i)) return;
        flag &f = flags[i];
        f.version = version;
        f.interploc = interpflagpos(f, f.interpangle);
        f.interptime = lastmillis;
        if(m_hold) conoutf(CON_GAMEINFO, "%s picked up the flag for %s", teamcolorname(d), teamcolor("your team", d->team, "the enemy team"));
        else if(m_protect || f.droptime) conoutf(CON_GAMEINFO, "%s picked up %s", teamcolorname(d), teamcolorflag(f));
        else conoutf(CON_GAMEINFO, "%s stole %s", teamcolorname(d), teamcolorflag(f));
        ownflag(i, d, lastmillis, m_hold ? ctfteamflag(d->team) : -1);
        teamsound(d, S_FLAGPICKUP);
    }

    void invisflag(int i, int invis)
    {
        if(!flags.inrange(i)) return;
        flag &f = flags[i];
        if(invis>0) f.vistime = 0;
        else if(!f.vistime) f.vistime = lastmillis;
    }

    void checkitems(fpsent *d)
    {
        if(d->state!=CS_ALIVE) return;
        vec o = d->feetpos();
        bool tookflag = false;
        loopv(flags)
        {
            flag &f = flags[i];
            if((m_hold ? f.spawnindex < 0 : !ctfflagteam(f.team) || f.team == ctfteamflag(d->team)) || f.owner || (f.droptime && f.droploc.x<0)) continue;
            const vec &loc = f.droptime ? f.droploc : f.spawnloc;
            if(o.dist(loc) < FLAGRADIUS)
            {
                if(d->flagpickup&(1<<f.id)) continue;
                if(m_hold || ((lookupmaterial(o)&MATF_CLIP) != MAT_GAMECLIP && (lookupmaterial(loc)&MATF_CLIP) != MAT_GAMECLIP))
                {
                    tookflag = true;
                    addmsg(N_TAKEFLAG, "rcii", d, i, f.version);
                }
                d->flagpickup |= 1<<f.id;
            }
            else d->flagpickup &= ~(1<<f.id);
        }
        if(m_hold) return;
        loopv(flags)
        {
            flag &f = flags[i];
            if(!ctfflagteam(f.team) || f.team != ctfteamflag(d->team) || f.owner || (f.droptime && f.droploc.x<0)) continue;
            const vec &loc = f.droptime ? f.droploc : f.spawnloc;
            if(o.dist(loc) < FLAGRADIUS)
            {
                if(!tookflag && d->flagpickup&(1<<f.id)) continue;
                if((lookupmaterial(o)&MATF_CLIP) != MAT_GAMECLIP && (lookupmaterial(loc)&MATF_CLIP) != MAT_GAMECLIP)
                    addmsg(N_TAKEFLAG, "rcii", d, i, f.version);
                d->flagpickup |= 1<<f.id;
            }
            else d->flagpickup &= ~(1<<f.id);
       }
    }

    void respawned(fpsent *d)
    {
        vec o = d->feetpos();
        d->flagpickup = 0;
        loopv(flags)
        {
            flag &f = flags[i];
            if((m_hold ? f.spawnindex < 0 : !ctfflagteam(f.team)) || f.owner || (f.droptime && f.droploc.x<0)) continue;
            if(o.dist(f.droptime ? f.droploc : f.spawnloc) < FLAGRADIUS) d->flagpickup |= 1<<f.id;
       }
    }

    int respawnwait(fpsent *d, int delay = 0)
    {
        return m_efficiency || !m_protect ? d->respawnwait(RESPAWNSECS, delay) : 0;
    }

    float rateholdspawn(const extentity &e)
    {   // avoid spawning too far away from active flag
        float minflagdist = 1e16f;
        loopv(flags)
        {
            const flag &f = flags[i];
            if(f.spawnindex < 0 || (!f.owner && (!f.droptime || f.droploc.x < 0))) continue;
            minflagdist = min(minflagdist, e.o.dist(f.owner ? f.owner->o : f.droploc));
        }
        return minflagdist < 1e16f ? proximityscore(minflagdist, 400.0f, 800.0f) : 1.0f;
    }

    float ratespawn(fpsent *d, const extentity &e)
    {
        return m_hold ? rateholdspawn(e) : 1.0f;
    }

    int getspawngroup(fpsent *d)
    {
        return m_hold ? 0 : ctfteamflag(d->team);
    }

	bool aihomerun(fpsent *d, ai::aistate &b)
	{
	    if(m_protect || m_hold)
	    {
            static vector<ai::interest> interests;
	        loopk(2)
	        {
                interests.setsize(0);
                ai::assist(d, b, interests, k != 0);
                if(ai::parseinterests(d, b, interests, false, true)) return true;
	        }
	    }
	    else
	    {
            vec pos = d->feetpos();
            loopk(2)
            {
                int goal = -1;
                loopv(flags)
                {
                    flag &g = flags[i];
                    if(g.team == ctfteamflag(d->team) && (k || (!g.owner && !g.droptime)) &&
                        (!flags.inrange(goal) || g.pos().squaredist(pos) < flags[goal].pos().squaredist(pos)))
                    {
                        goal = i;
                    }
                }
                if(flags.inrange(goal) && ai::makeroute(d, b, flags[goal].pos()))
                {
                    d->ai->switchstate(b, ai::AI_S_PURSUE, ai::AI_T_AFFINITY, goal);
                    return true;
                }
            }
	    }
	    if(b.type == ai::AI_S_INTEREST && b.targtype == ai::AI_T_NODE) return true; // we already did this..
		if(randomnode(d, b, ai::SIGHTMIN, 1e16f))
		{
            d->ai->switchstate(b, ai::AI_S_INTEREST, ai::AI_T_NODE, d->ai->route[0]);
            return true;
		}
		return false;
	}

	bool aicheck(fpsent *d, ai::aistate &b)
	{
        static vector<int> takenflags;
        takenflags.setsize(0);
        loopv(flags)
        {
            flag &g = flags[i];
            if(g.owner == d) return aihomerun(d, b);
            else if(g.team == ctfteamflag(d->team) && ((g.owner && g.team != ctfteamflag(g.owner->team)) || g.droptime))
                takenflags.add(i);
        }
        if(!ai::badhealth(d) && !takenflags.empty())
        {
            int flag = takenflags.length() > 2 ? rnd(takenflags.length()) : 0;
            d->ai->switchstate(b, ai::AI_S_PURSUE, ai::AI_T_AFFINITY, takenflags[flag]);
            return true;
        }
		return false;
	}

	void aifind(fpsent *d, ai::aistate &b, vector<ai::interest> &interests)
	{
		vec pos = d->feetpos();
		loopvj(flags)
		{
			flag &f = flags[j];
			if((!m_protect && !m_hold) || f.owner != d)
			{
				static vector<int> targets; // build a list of others who are interested in this
				targets.setsize(0);
				bool home = !m_hold && f.team == ctfteamflag(d->team);
				ai::checkothers(targets, d, home ? ai::AI_S_DEFEND : ai::AI_S_PURSUE, ai::AI_T_AFFINITY, j, true);
				fpsent *e = NULL;
				loopi(numdynents()) if((e = (fpsent *)iterdynents(i)) && !e->ai && e->state == CS_ALIVE && isteam(d->team, e->team))
				{ // try to guess what non ai are doing
					vec ep = e->feetpos();
					if(targets.find(e->clientnum) < 0 && (ep.squaredist(f.pos()) <= (FLAGRADIUS*FLAGRADIUS*4) || f.owner == e))
						targets.add(e->clientnum);
				}
				if(home)
				{
					bool guard = false;
					if((f.owner && f.team != ctfteamflag(f.owner->team)) || f.droptime || targets.empty()) guard = true;
					else if(d->hasammo(d->ai->weappref))
					{ // see if we can relieve someone who only has a piece of crap
						fpsent *t;
						loopvk(targets) if((t = getclient(targets[k])))
						{
							if((t->ai && !t->hasammo(t->ai->weappref)) || (!t->ai && (t->gunselect == GUN_FIST || t->gunselect == GUN_PISTOL)))
							{
								guard = true;
								break;
							}
						}
					}
					if(guard)
					{ // defend the flag
						ai::interest &n = interests.add();
						n.state = ai::AI_S_DEFEND;
						n.node = ai::closestwaypoint(f.pos(), ai::SIGHTMIN, true);
						n.target = j;
						n.targtype = ai::AI_T_AFFINITY;
						n.score = pos.squaredist(f.pos())/100.f;
					}
				}
				else
				{
					if(targets.empty())
					{ // attack the flag
						ai::interest &n = interests.add();
						n.state = ai::AI_S_PURSUE;
						n.node = ai::closestwaypoint(f.pos(), ai::SIGHTMIN, true);
						n.target = j;
						n.targtype = ai::AI_T_AFFINITY;
						n.score = pos.squaredist(f.pos());
					}
					else
					{ // help by defending the attacker
						fpsent *t;
						loopvk(targets) if((t = getclient(targets[k])))
						{
							ai::interest &n = interests.add();
							n.state = ai::AI_S_DEFEND;
							n.node = t->lastnode;
							n.target = t->clientnum;
							n.targtype = ai::AI_T_PLAYER;
							n.score = d->o.squaredist(t->o);
						}
					}
				}
			}
		}
	}

	bool aidefend(fpsent *d, ai::aistate &b)
	{
        loopv(flags)
        {
            flag &g = flags[i];
            if(g.owner == d) return aihomerun(d, b);
        }
		if(flags.inrange(b.target))
		{
			flag &f = flags[b.target];
			if(f.droptime) return ai::makeroute(d, b, f.pos());
			if(f.owner) return ai::violence(d, b, f.owner, 4);
			int walk = 0;
			if(lastmillis-b.millis >= (201-d->skill)*33)
			{
				static vector<int> targets; // build a list of others who are interested in this
				targets.setsize(0);
				ai::checkothers(targets, d, ai::AI_S_DEFEND, ai::AI_T_AFFINITY, b.target, true);
				fpsent *e = NULL;
				loopi(numdynents()) if((e = (fpsent *)iterdynents(i)) && !e->ai && e->state == CS_ALIVE && isteam(d->team, e->team))
				{ // try to guess what non ai are doing
					vec ep = e->feetpos();
					if(targets.find(e->clientnum) < 0 && (ep.squaredist(f.pos()) <= (FLAGRADIUS*FLAGRADIUS*4) || f.owner == e))
						targets.add(e->clientnum);
				}
				if(!targets.empty())
				{
					d->ai->trywipe = true; // re-evaluate so as not to herd
					return true;
				}
				else
				{
					walk = 2;
					b.millis = lastmillis;
				}
			}
			vec pos = d->feetpos();
			float mindist = float(FLAGRADIUS*FLAGRADIUS*8);
			loopv(flags)
			{ // get out of the way of the returnee!
				flag &g = flags[i];
				if(pos.squaredist(g.pos()) <= mindist)
				{
					if(!m_protect && !m_hold && g.owner && !strcmp(g.owner->team, d->team)) walk = 1;
					if(g.droptime && ai::makeroute(d, b, g.pos())) return true;
				}
			}
			return ai::defend(d, b, f.pos(), float(FLAGRADIUS*2), float(FLAGRADIUS*(2+(walk*2))), walk);
		}
		return false;
	}

	bool aipursue(fpsent *d, ai::aistate &b)
	{
		if(flags.inrange(b.target))
		{
			flag &f = flags[b.target];
            if(f.owner == d) return aihomerun(d, b);
			if(!m_hold && f.team == ctfteamflag(d->team))
			{
				if(f.droptime) return ai::makeroute(d, b, f.pos());
				if(f.owner) return ai::violence(d, b, f.owner, 4);
                loopv(flags)
                {
                    flag &g = flags[i];
                    if(g.owner == d) return ai::makeroute(d, b, f.pos());
                }
			}
			else
			{
				if(f.owner) return ai::violence(d, b, f.owner, 4);
				return ai::makeroute(d, b, f.pos());
			}
		}
		return false;
	}
};

extern ctfclientmode ctfmode;
ICOMMAND(dropflag, "", (), { ctfmode.trydropflag(); });

#endif

#elif SERVMODE

case N_TRYDROPFLAG:
{
    if((ci->state.state!=CS_SPECTATOR || ci->local || ci->privilege) && cq && smode==&ctfmode) ctfmode.dropflag(cq);
    break;
}

case N_TAKEFLAG:
{
    int flag = getint(p), version = getint(p);
    if((ci->state.state!=CS_SPECTATOR || ci->local || ci->privilege) && cq && smode==&ctfmode) ctfmode.takeflag(cq, flag, version);
    break;
}

case N_INITFLAGS:
    if(smode==&ctfmode) ctfmode.parseflags(p, (ci->state.state!=CS_SPECTATOR || ci->privilege || ci->local) && !strcmp(ci->clientmap, smapname));
    break;

#else

case N_INITFLAGS:
{
    ctfmode.parseflags(p, m_ctf);
    break;
}

case N_DROPFLAG:
{
    int ocn = getint(p), flag = getint(p), version = getint(p);
    vec droploc;
    loopk(3) droploc[k] = getint(p)/DMF;
    fpsent *o = ocn==player1->clientnum ? player1 : newclient(ocn);
    if(o && m_ctf) ctfmode.dropflag(o, flag, version, droploc);
    break;
}

case N_SCOREFLAG:
{
    int ocn = getint(p), relayflag = getint(p), relayversion = getint(p), goalflag = getint(p), goalversion = getint(p), goalspawn = getint(p), team = getint(p), score = getint(p), oflags = getint(p);
    fpsent *o = ocn==player1->clientnum ? player1 : newclient(ocn);
    if(o && m_ctf) ctfmode.scoreflag(o, relayflag, relayversion, goalflag, goalversion, goalspawn, team, score, oflags);
    break;
}

case N_RETURNFLAG:
{
    int ocn = getint(p), flag = getint(p), version = getint(p);
    fpsent *o = ocn==player1->clientnum ? player1 : newclient(ocn);
    if(o && m_ctf) ctfmode.returnflag(o, flag, version);
    break;
}

case N_TAKEFLAG:
{
    int ocn = getint(p), flag = getint(p), version = getint(p);
    fpsent *o = ocn==player1->clientnum ? player1 : newclient(ocn);
    if(o && m_ctf) ctfmode.takeflag(o, flag, version);
    break;
}

case N_RESETFLAG:
{
    int flag = getint(p), version = getint(p), spawnindex = getint(p), team = getint(p), score = getint(p);
    if(m_ctf) ctfmode.resetflag(flag, version, spawnindex, team, score);
    break;
}

case N_INVISFLAG:
{
    int flag = getint(p), invis = getint(p);
    if(m_ctf) ctfmode.invisflag(flag, invis);
    break;
}

#endif

