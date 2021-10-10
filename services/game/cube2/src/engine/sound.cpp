// sound.cpp: basic positional sound using sdl_mixer

#include "engine.h"
#include "SDL_mixer.h"

bool nosound = true;

struct soundsample
{
    char *name;
    Mix_Chunk *chunk;

    soundsample() : name(NULL), chunk(NULL) {}
    ~soundsample() { DELETEA(name); }

    void cleanup() { if(chunk) { Mix_FreeChunk(chunk); chunk = NULL; } }
    bool load(bool msg = false);
};

struct soundslot
{
    soundsample *sample;
    int volume;
};

struct soundconfig
{
    int slots, numslots;
    int maxuses;

    bool hasslot(const soundslot *p, const vector<soundslot> &v) const
    {
        return p >= v.getbuf() + slots && p < v.getbuf() + slots+numslots && slots+numslots < v.length(); 
    }

    int chooseslot(int flags) const
    {
        if(flags&SND_NO_ALT || numslots <= 1) return slots;
        if(flags&SND_USE_ALT) return slots + 1 + rnd(numslots - 1);
        return slots + rnd(numslots);
    }
};

struct soundchannel
{ 
    int id;
    bool inuse;
    vec loc; 
    soundslot *slot;
    extentity *ent; 
    int radius, volume, pan, flags;
    bool dirty;

    soundchannel(int id) : id(id) { reset(); }

    bool hasloc() const { return loc.x >= -1e15f; }
    void clearloc() { loc = vec(-1e16f, -1e16f, -1e16f); }

    void reset()
    {
        inuse = false;
        clearloc();
        slot = NULL;
        ent = NULL;
        radius = 0;
        volume = -1;
        pan = -1;
        flags = 0;
        dirty = false;
    }
};
vector<soundchannel> channels;
int maxchannels = 0;

soundchannel &newchannel(int n, soundslot *slot, const vec *loc = NULL, extentity *ent = NULL, int flags = 0, int radius = 0)
{
    if(ent)
    {
        loc = &ent->o;
        ent->flags |= EF_SOUND;
    }
    while(!channels.inrange(n)) channels.add(channels.length());
    soundchannel &chan = channels[n];
    chan.reset();
    chan.inuse = true;
    if(loc) chan.loc = *loc;
    chan.slot = slot;
    chan.ent = ent;
    chan.flags = 0;
    chan.radius = radius;
    return chan;
}

void freechannel(int n)
{
    if(!channels.inrange(n) || !channels[n].inuse) return;
    soundchannel &chan = channels[n];
    chan.inuse = false;
    if(chan.ent) chan.ent->flags &= ~EF_SOUND;
}

void syncchannel(soundchannel &chan)
{
    if(!chan.dirty) return;
    if(!Mix_FadingChannel(chan.id)) Mix_Volume(chan.id, chan.volume);
    Mix_SetPanning(chan.id, 255-chan.pan, chan.pan);
    chan.dirty = false;
}

void stopchannels()
{
    loopv(channels)
    {
        soundchannel &chan = channels[i];
        if(!chan.inuse) continue;
        Mix_HaltChannel(i);
        freechannel(i);
    }
}

void setmusicvol(int musicvol);
extern int musicvol;
static int curvol = 0;
VARFP(soundvol, 0, 255, 255,
{
    if(!soundvol) { stopchannels(); setmusicvol(0); }
    else if(!curvol) setmusicvol(musicvol);
    curvol = soundvol;
});
VARFP(musicvol, 0, 128, 255, setmusicvol(soundvol ? musicvol : 0));

char *musicfile = NULL, *musicdonecmd = NULL;

Mix_Music *music = NULL;
SDL_RWops *musicrw = NULL;
stream *musicstream = NULL;

void setmusicvol(int musicvol)
{
    if(nosound) return;
    if(music) Mix_VolumeMusic((musicvol*MIX_MAX_VOLUME)/255);
}

void stopmusic()
{
    if(nosound) return;
    DELETEA(musicfile);
    DELETEA(musicdonecmd);
    if(music)
    {
        Mix_HaltMusic();
        Mix_FreeMusic(music);
        music = NULL;
    }
    if(musicrw) { SDL_FreeRW(musicrw); musicrw = NULL; }
    DELETEP(musicstream);
}

#ifdef WIN32
#define AUDIODRIVER "directsound winmm"
#else
#define AUDIODRIVER ""
#endif
bool shouldinitaudio = true;
SVARF(audiodriver, AUDIODRIVER, { shouldinitaudio = true; initwarning("sound configuration", INIT_RESET, CHANGE_SOUND); });
VARF(usesound, 0, 1, 1, { shouldinitaudio = true; initwarning("sound configuration", INIT_RESET, CHANGE_SOUND); });
VARF(soundchans, 1, 32, 128, initwarning("sound configuration", INIT_RESET, CHANGE_SOUND));
VARF(soundfreq, 0, MIX_DEFAULT_FREQUENCY, 48000, initwarning("sound configuration", INIT_RESET, CHANGE_SOUND));
VARF(soundbufferlen, 128, 1024, 4096, initwarning("sound configuration", INIT_RESET, CHANGE_SOUND));

bool initaudio()
{
    static string fallback = "";
    static bool initfallback = true;
    static bool restorefallback = false;
    if(initfallback)
    {
        initfallback = false;
        if(char *env = SDL_getenv("SDL_AUDIODRIVER")) copystring(fallback, env);
    }
    if(!fallback[0] && audiodriver[0])
    {
        vector<char*> drivers;
        explodelist(audiodriver, drivers);
        loopv(drivers)
        {
            restorefallback = true;
            SDL_setenv("SDL_AUDIODRIVER", drivers[i], 1);
            if(SDL_InitSubSystem(SDL_INIT_AUDIO) >= 0)
            {
                drivers.deletearrays();
                return true;
            }
        }
        drivers.deletearrays();
    }
    if(restorefallback)
    {
        restorefallback = false;
    #ifdef WIN32
        SDL_setenv("SDL_AUDIODRIVER", fallback, 1);
    #else
        unsetenv("SDL_AUDIODRIVER");
    #endif
    }
    if(SDL_InitSubSystem(SDL_INIT_AUDIO) >= 0) return true;
    conoutf(CON_ERROR, "sound init failed: %s", SDL_GetError());
    return false;
}

void initsound()
{
    SDL_version version;
    SDL_GetVersion(&version);
    if(version.major == 2 && version.minor == 0 && version.patch == 6)
    {
        nosound = true;
        if(usesound) conoutf(CON_ERROR, "audio is broken in SDL 2.0.6");
        return;
    }

    if(shouldinitaudio)
    {
        shouldinitaudio = false;
        if(SDL_WasInit(SDL_INIT_AUDIO)) SDL_QuitSubSystem(SDL_INIT_AUDIO);
        if(!usesound || !initaudio())
        {
            nosound = true;
            return;
        }
    }

    if(Mix_OpenAudio(soundfreq, MIX_DEFAULT_FORMAT, 2, soundbufferlen)<0)
    {
        nosound = true;
        conoutf(CON_ERROR, "sound init failed (SDL_mixer): %s", Mix_GetError());
        return;
    }
	Mix_AllocateChannels(soundchans);	
    maxchannels = soundchans;
    nosound = false;
}

void musicdone()
{
    if(music) { Mix_HaltMusic(); Mix_FreeMusic(music); music = NULL; }
    if(musicrw) { SDL_FreeRW(musicrw); musicrw = NULL; }
    DELETEP(musicstream);
    DELETEA(musicfile);
    if(!musicdonecmd) return;
    char *cmd = musicdonecmd;
    musicdonecmd = NULL;
    execute(cmd);
    delete[] cmd;
}

Mix_Music *loadmusic(const char *name)
{
    if(!musicstream) musicstream = openzipfile(name, "rb");
    if(musicstream)
    {
        if(!musicrw) musicrw = musicstream->rwops();
        if(!musicrw) DELETEP(musicstream);
    }
    if(musicrw) music = Mix_LoadMUSType_RW(musicrw, MUS_NONE, 0);
    else music = Mix_LoadMUS(findfile(name, "rb")); 
    if(!music)
    {
        if(musicrw) { SDL_FreeRW(musicrw); musicrw = NULL; }
        DELETEP(musicstream);
    }
    return music;
}
 
void startmusic(char *name, char *cmd)
{
    if(nosound) return;
    stopmusic();
    if(soundvol && musicvol && *name)
    {
        defformatstring(file, "packages/%s", name);
        path(file);
        if(loadmusic(file))
        {
            DELETEA(musicfile);
            DELETEA(musicdonecmd);
            musicfile = newstring(file);
            if(cmd[0]) musicdonecmd = newstring(cmd);
            Mix_PlayMusic(music, cmd[0] ? 0 : -1);
            Mix_VolumeMusic((musicvol*MIX_MAX_VOLUME)/255);
            intret(1);
        }
        else
        {
            conoutf(CON_ERROR, "could not play music: %s", file);
            intret(0); 
        }
    }
}

COMMANDN(music, startmusic, "ss");

static Mix_Chunk *loadwav(const char *name)
{
    Mix_Chunk *c = NULL;
    stream *z = openzipfile(name, "rb");
    if(z)
    {
        SDL_RWops *rw = z->rwops();
        if(rw)
        {
            c = Mix_LoadWAV_RW(rw, 0);
            SDL_FreeRW(rw);
        }
        delete z;
    }
    if(!c) c = Mix_LoadWAV(findfile(name, "rb"));
    return c;
}

template<class T> static void scalewav(T* dst, T* src, size_t len, int scale)
{
    len /= sizeof(T);
    const T* end = src + len;
    if(scale==2) for(; src < end; src++, dst += scale)
    {
        T s = src[0];
        dst[0] = s;
        dst[1] = s;
    }
    else if(scale==4) for(; src < end; src++, dst += scale)
    {
        T s = src[0];
        dst[0] = s;
        dst[1] = s;
        dst[2] = s;
        dst[3] = s;
    }
    else for(; src < end; src++)
    {
        T s = src[0];
        loopi(scale) *dst++ = s;
    }
}

static Mix_Chunk *loadwavscaled(const char *name)
{
    int mixerfreq = 0;
    Uint16 mixerformat = 0;
    int mixerchannels = 0;
    if(!Mix_QuerySpec(&mixerfreq, &mixerformat, &mixerchannels)) return NULL;

    SDL_AudioSpec spec;
    Uint8 *audiobuf = NULL;
    Uint32 audiolen = 0;
    stream *z = openzipfile(name, "rb");
    if(z)
    {
        SDL_RWops *rw = z->rwops();
        if(rw)
        {
            SDL_LoadWAV_RW(rw, 0, &spec, &audiobuf, &audiolen);
            SDL_FreeRW(rw);
        }
        delete z;
    }
    if(!audiobuf) SDL_LoadWAV(findfile(name, "rb"), &spec, &audiobuf, &audiolen);
    if(!audiobuf) return NULL;
    int samplesize = ((spec.format&0xFF)/8) * spec.channels;
    int scale = mixerfreq / spec.freq;
    if(scale >= 2)
    {
        Uint8 *scalebuf = (Uint8*)SDL_malloc(audiolen * scale);
        if(scalebuf)
        {
            switch(samplesize)
            {
                case 1: scalewav((uchar*)scalebuf, (uchar*)audiobuf, audiolen, scale); break;
                case 2: scalewav((ushort*)scalebuf, (ushort*)audiobuf, audiolen, scale); break;
                case 4: scalewav((uint*)scalebuf, (uint*)audiobuf, audiolen, scale); break;
                case 8: scalewav((ullong*)scalebuf, (ullong*)audiobuf, audiolen, scale); break;
                default: SDL_free(scalebuf); scalebuf = NULL; break;
            }
            if(scalebuf)
            {
                SDL_free(audiobuf);
                audiobuf = scalebuf;
                audiolen *= scale;
                spec.freq *= scale;
            }
        }
    }
    if(spec.freq != mixerfreq || spec.format != mixerformat || spec.channels != mixerchannels)
    {
        SDL_AudioCVT cvt;
        if(SDL_BuildAudioCVT(&cvt, spec.format, spec.channels, spec.freq, mixerformat, mixerchannels, mixerfreq) < 0)
        {
            SDL_free(audiobuf);
            return NULL;
        }
        if(cvt.filters[0])
        {
            cvt.len = audiolen & ~(samplesize-1);
            cvt.buf = (Uint8*)SDL_malloc(cvt.len * cvt.len_mult);
            if(!cvt.buf) { SDL_free(audiobuf); return NULL; }
            SDL_memcpy(cvt.buf, audiobuf, cvt.len);
            SDL_free(audiobuf);
            if(SDL_ConvertAudio(&cvt) < 0) { SDL_free(cvt.buf); return NULL; }
            audiobuf = cvt.buf;
            audiolen = cvt.len_cvt;
        }
    }
    Mix_Chunk *c = Mix_QuickLoad_RAW(audiobuf, audiolen);
    if(!c) { SDL_free(audiobuf); return NULL; }
    c->allocated = 1;
    return c;
}

VARFP(fixwav, 0, 1, 1, initwarning("sound configuration", INIT_LOAD, CHANGE_SOUND));

bool soundsample::load(bool msg)
{
    if(chunk) return true;
    if(!name[0]) return false;

    static const char * const exts[] = { "", ".wav", ".ogg" };
    string filename;
    loopi(sizeof(exts)/sizeof(exts[0]))
    {
        formatstring(filename, "packages/sounds/%s%s", name, exts[i]);
        if(msg && !i) renderprogress(0, filename);
        path(filename);
        if(fixwav)
        {
            size_t len = strlen(filename);
            if(len >= 4 && !strcasecmp(filename + len - 4, ".wav"))
            {
                chunk = loadwavscaled(filename);
                if(chunk) return true;
            }
        }
        chunk = loadwav(filename);
        if(chunk) return true;
    }

    conoutf(CON_ERROR, "failed to load sample: packages/sounds/%s", name);
    return false;
}

static hashnameset<soundsample> samples;

static void cleanupsamples()
{
    enumerate(samples, soundsample, s, s.cleanup());
}

static struct soundtype
{
    vector<soundslot> slots;
    vector<soundconfig> configs;

    int findsound(const char *name, int vol)
    {
        loopv(configs)
        {
            soundconfig &s = configs[i];
            loopj(s.numslots)
            {
                soundslot &c = slots[s.slots+j];
                if(!strcmp(c.sample->name, name) && (!vol || c.volume==vol)) return i;
            }
        }
        return -1;
    }

    int addslot(const char *name, int vol)
    {
        soundsample *s = samples.access(name);
        if(!s)
        {
            char *n = newstring(name);
            s = &samples[n];
            s->name = n;
            s->chunk = NULL;
        }
        soundslot *oldslots = slots.getbuf();
        int oldlen = slots.length();
        soundslot &slot = slots.add();
        // soundslots.add() may relocate slot pointers
        if(slots.getbuf() != oldslots) loopv(channels)
        {
            soundchannel &chan = channels[i];
            if(chan.inuse && chan.slot >= oldslots && chan.slot < &oldslots[oldlen])
                chan.slot = &slots[chan.slot - oldslots];
        }
        slot.sample = s;
        slot.volume = vol ? vol : 100;
        return oldlen;
    }

    int addsound(const char *name, int vol, int maxuses = 0)
    {
        soundconfig &s = configs.add();
        s.slots = addslot(name, vol);
        s.numslots = 1;
        s.maxuses = maxuses;
        return configs.length()-1;
    }

    void addalt(const char *name, int vol)
    {
        if(configs.empty()) return;
        addslot(name, vol);
        configs.last().numslots++;
    }

    void clear()
    {
        slots.setsize(0);
        configs.setsize(0);
    }

    void reset()
    {
        loopv(channels)
        {
            soundchannel &chan = channels[i];
            if(chan.inuse && slots.inbuf(chan.slot))
            {
                Mix_HaltChannel(i);
                freechannel(i);
            }
        }
        clear();
    }

    void preloadsound(int n)
    {
        if(nosound || !configs.inrange(n)) return;
        soundconfig &config = configs[n];
        loopk(config.numslots) slots[config.slots+k].sample->load(true);
    }

    bool playing(const soundchannel &chan, const soundconfig &config) const
    {
        return chan.inuse && config.hasslot(chan.slot, slots);
    }
} gamesounds, mapsounds;

void registersound(char *name, int *vol) { intret(gamesounds.addsound(name, *vol, 0)); }
COMMAND(registersound, "si");

void mapsound(char *name, int *vol, int *maxuses) { intret(mapsounds.addsound(name, *vol, *maxuses < 0 ? 0 : max(1, *maxuses))); }
COMMAND(mapsound, "sii");

void altsound(char *name, int *vol) { gamesounds.addalt(name, *vol); }
COMMAND(altsound, "si");

void altmapsound(char *name, int *vol) { mapsounds.addalt(name, *vol); }
COMMAND(altmapsound, "si");

ICOMMAND(numsounds, "", (), intret(gamesounds.configs.length()));
ICOMMAND(nummapsounds, "", (), intret(mapsounds.configs.length()));

void soundreset()
{
    gamesounds.reset();
}
COMMAND(soundreset, "");

void mapsoundreset()
{
    mapsounds.reset();
}
COMMAND(mapsoundreset, "");

void resetchannels()
{
    loopv(channels) if(channels[i].inuse) freechannel(i);
    channels.shrink(0);
}

void clear_sound()
{
    closemumble();
    if(nosound) return;
    stopmusic();

    cleanupsamples();
    gamesounds.clear();
    mapsounds.clear();
    samples.clear();
    Mix_CloseAudio();
    resetchannels();
}

void stopmapsounds()
{
    loopv(channels) if(channels[i].inuse && channels[i].ent)
    {
        Mix_HaltChannel(i);
        freechannel(i);
    }
}

void clearmapsounds()
{
    stopmapsounds();
    mapsounds.clear();
}

void stopmapsound(extentity *e)
{
    loopv(channels)
    {
        soundchannel &chan = channels[i];
        if(chan.inuse && chan.ent == e)
        {
            Mix_HaltChannel(i);
            freechannel(i);
        }
    }
}

void checkmapsounds()
{
    const vector<extentity *> &ents = entities::getents();
    loopv(ents)
    {
        extentity &e = *ents[i];
        if(e.type!=ET_SOUND) continue;
        if(camera1->o.dist(e.o) < e.attr2)
        {
            if(!(e.flags&EF_SOUND)) playsound(e.attr1, NULL, &e, SND_MAP, -1);
        }
        else if(e.flags&EF_SOUND) stopmapsound(&e);
    }
}

VAR(stereo, 0, 1, 1);

bool updatechannel(soundchannel &chan)
{
    if(!chan.slot) return false;
    int vol = soundvol, pan = 255/2;
    if(chan.hasloc())
    {
        vec v;
        float dist = chan.loc.dist(camera1->o, v);
        int rad = 0;
        if(chan.ent)
        {
            rad = chan.ent->attr2;
            if(chan.ent->attr3)
            {
                rad -= chan.ent->attr3;
                dist -= chan.ent->attr3;
            }
        }
        else if(chan.radius > 0) rad = chan.radius;
        if(rad > 0) vol -= int(clamp(dist/rad, 0.0f, 1.0f)*soundvol); // simple mono distance attenuation
        if(stereo && (v.x != 0 || v.y != 0) && dist>0)
        {
            v.rotate_around_z(-camera1->yaw*RAD);
            pan = int(255.9f*(0.5f - 0.5f*v.x/v.magnitude2())); // range is from 0 (left) to 255 (right)
        }
    }
    vol = (vol*MIX_MAX_VOLUME*chan.slot->volume)/255/255;
    vol = min(vol, MIX_MAX_VOLUME);
    if(vol == chan.volume && pan == chan.pan) return false;
    chan.volume = vol;
    chan.pan = pan;
    chan.dirty = true;
    return true;
}  

void reclaimchannels()
{
    loopv(channels)
    {
        soundchannel &chan = channels[i];
        if(chan.inuse && !Mix_Playing(i)) freechannel(i);
    }
}

void syncchannels()
{
    loopv(channels)
    {
        soundchannel &chan = channels[i];
        if(chan.inuse && chan.hasloc() && updatechannel(chan)) syncchannel(chan);
    }
}

VARP(minimizedsounds, 0, 0, 1);

void updatesounds()
{
    updatemumble();
    if(nosound) return;
    if(minimized && !minimizedsounds) stopsounds();
    else
    {
        reclaimchannels();
        if(mainmenu) stopmapsounds();
        else checkmapsounds();
        syncchannels();
    }
    if(music)
    {
        if(!Mix_PlayingMusic()) musicdone();
        else if(Mix_PausedMusic()) Mix_ResumeMusic();
    }
}

VARP(maxsoundsatonce, 0, 7, 100);

VAR(dbgsound, 0, 0, 1);

void preloadsound(int n)
{
    gamesounds.preloadsound(n);
}

void preloadmapsound(int n)
{
    mapsounds.preloadsound(n);
}

void preloadmapsounds()
{
    const vector<extentity *> &ents = entities::getents();
    loopv(ents)
    {
        extentity &e = *ents[i];
        if(e.type==ET_SOUND) mapsounds.preloadsound(e.attr1);
    }
}
 
int playsound(int n, const vec *loc, extentity *ent, int flags, int loops, int fade, int chanid, int radius, int expire)
{
    if(nosound || !soundvol || (minimized && !minimizedsounds)) return -1;

    soundtype &sounds = ent || flags&SND_MAP ? mapsounds : gamesounds;
    if(!sounds.configs.inrange(n)) { conoutf(CON_WARN, "unregistered sound: %d", n); return -1; }
    soundconfig &config = sounds.configs[n];

    if(loc)
    {
        // cull sounds that are unlikely to be heard
        int maxrad = game::maxsoundradius(n);
        if(radius <= 0 || maxrad < radius) radius = maxrad;
        if(camera1->o.dist(*loc) > 1.5f*radius)
        {
            if(channels.inrange(chanid) && sounds.playing(channels[chanid], config))
            {
                Mix_HaltChannel(chanid);
                freechannel(chanid);
            }
            return -1;    
        }
    }

    if(chanid < 0)
    {
        if(config.maxuses)
        {
            int uses = 0;
            loopv(channels) if(sounds.playing(channels[i], config) && ++uses >= config.maxuses) return -1;
        }

        // avoid bursts of sounds with heavy packetloss and in sp
        static int soundsatonce = 0, lastsoundmillis = 0;
        if(totalmillis == lastsoundmillis) soundsatonce++; else soundsatonce = 1;
        lastsoundmillis = totalmillis;
        if(maxsoundsatonce && soundsatonce > maxsoundsatonce) return -1;
    }

    if(channels.inrange(chanid))
    {
        soundchannel &chan = channels[chanid];
        if(sounds.playing(chan, config))
        {
            if(loc) chan.loc = *loc;
            else if(chan.hasloc()) chan.clearloc();
            return chanid;
        }
    }
    if(fade < 0) return -1;

    soundslot &slot = sounds.slots[config.chooseslot(flags)];
    if(!slot.sample->chunk && !slot.sample->load()) return -1;

    if(dbgsound) conoutf(CON_DEBUG, "sound: %s", slot.sample->name);
 
    chanid = -1;
    loopv(channels) if(!channels[i].inuse) { chanid = i; break; }
    if(chanid < 0 && channels.length() < maxchannels) chanid = channels.length();
    if(chanid < 0) loopv(channels) if(!channels[i].volume) { chanid = i; break; }
    if(chanid < 0) return -1;

    soundchannel &chan = newchannel(chanid, &slot, loc, ent, flags, radius);
    updatechannel(chan);
    int playing = expire >= 0 ? Mix_PlayChannelTimed(chanid, slot.sample->chunk, loops, expire) : Mix_PlayChannel(chanid, slot.sample->chunk, loops);
    if(playing >= 0) syncchannel(chan); 
    else freechannel(chanid);
    return playing;
}

void stopsounds()
{
    loopv(channels) if(channels[i].inuse)
    {
        Mix_HaltChannel(i);
        freechannel(i);
    }
}

bool stopsound(int n, int chanid, int fade)
{
    if(!gamesounds.configs.inrange(n) || !channels.inrange(chanid) || !channels[chanid].inuse || !gamesounds.playing(channels[chanid], gamesounds.configs[n])) return false;
    if(dbgsound) conoutf(CON_DEBUG, "stopsound: %s", channels[chanid].slot->sample->name);
    if(!fade || !Mix_FadeOutChannel(chanid, fade))
    {
        Mix_HaltChannel(chanid);
        freechannel(chanid);
    }
    return true;
}

int playsoundname(const char *s, const vec *loc, int vol, int flags, int loops, int fade, int chanid, int radius, int expire) 
{ 
    if(!vol) vol = 100;
    int id = gamesounds.findsound(s, vol);
    if(id < 0) id = gamesounds.addsound(s, vol);
    return playsound(id, loc, NULL, flags, loops, fade, chanid, radius, expire);
}

extern "C" {

void bb_playsoundname(const char *s, float x, float y, float z)
{
    static vec temp;
    temp.x = x; temp.y = y; temp.z = z;
    playsoundname(s, &temp);
}

}

void sound(int *n) { playsound(*n); }
COMMAND(sound, "i");

void resetsound()
{
    clearchanges(CHANGE_SOUND);
    if(!nosound) 
    {
        cleanupsamples();
        if(music)
        {
            Mix_HaltMusic();
            Mix_FreeMusic(music);
        }
        if(musicstream) musicstream->seek(0, SEEK_SET);
        Mix_CloseAudio();
    }
    initsound();
    resetchannels();
    if(nosound)
    {
        DELETEA(musicfile);
        DELETEA(musicdonecmd);
        music = NULL;
        cleanupsamples();
        return;
    }
    if(music && loadmusic(musicfile))
    {
        Mix_PlayMusic(music, musicdonecmd ? 0 : -1);
        Mix_VolumeMusic((musicvol*MIX_MAX_VOLUME)/255);
    }
    else
    {
        DELETEA(musicfile);
        DELETEA(musicdonecmd);
    }
}

COMMAND(resetsound, "");

#ifdef WIN32

#include <wchar.h>

#else

#include <unistd.h>

#ifdef _POSIX_SHARED_MEMORY_OBJECTS
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <fcntl.h>
#include <wchar.h>
#endif

#endif

#if defined(WIN32) || defined(_POSIX_SHARED_MEMORY_OBJECTS)
struct MumbleInfo
{
    int version, timestamp;
    vec pos, front, top;
    wchar_t name[256];
};
#endif

#ifdef WIN32
static HANDLE mumblelink = NULL;
static MumbleInfo *mumbleinfo = NULL;
#define VALID_MUMBLELINK (mumblelink && mumbleinfo)
#elif defined(_POSIX_SHARED_MEMORY_OBJECTS)
static int mumblelink = -1;
static MumbleInfo *mumbleinfo = (MumbleInfo *)-1; 
#define VALID_MUMBLELINK (mumblelink >= 0 && mumbleinfo != (MumbleInfo *)-1)
#endif

#ifdef VALID_MUMBLELINK
VARFP(mumble, 0, 1, 1, { if(mumble) initmumble(); else closemumble(); });
#else
VARFP(mumble, 0, 0, 1, { if(mumble) initmumble(); else closemumble(); });
#endif

void initmumble()
{
    if(!mumble) return;
#ifdef VALID_MUMBLELINK
    if(VALID_MUMBLELINK) return;

    #ifdef WIN32
        mumblelink = OpenFileMapping(FILE_MAP_ALL_ACCESS, FALSE, "MumbleLink");
        if(mumblelink)
        {
            mumbleinfo = (MumbleInfo *)MapViewOfFile(mumblelink, FILE_MAP_ALL_ACCESS, 0, 0, sizeof(MumbleInfo));
            if(mumbleinfo) wcsncpy(mumbleinfo->name, L"Sauerbraten", 256);
        }
    #elif defined(_POSIX_SHARED_MEMORY_OBJECTS)
        defformatstring(shmname, "/MumbleLink.%d", getuid());
        mumblelink = shm_open(shmname, O_RDWR, 0);
        if(mumblelink >= 0)
        {
            mumbleinfo = (MumbleInfo *)mmap(NULL, sizeof(MumbleInfo), PROT_READ|PROT_WRITE, MAP_SHARED, mumblelink, 0);
            if(mumbleinfo != (MumbleInfo *)-1) wcsncpy(mumbleinfo->name, L"Sauerbraten", 256);
        }
    #endif
    if(!VALID_MUMBLELINK) closemumble();
#else
    conoutf(CON_ERROR, "Mumble positional audio is not available on this platform.");
#endif
}

void closemumble()
{
#ifdef WIN32
    if(mumbleinfo) { UnmapViewOfFile(mumbleinfo); mumbleinfo = NULL; }
    if(mumblelink) { CloseHandle(mumblelink); mumblelink = NULL; }
#elif defined(_POSIX_SHARED_MEMORY_OBJECTS)
    if(mumbleinfo != (MumbleInfo *)-1) { munmap(mumbleinfo, sizeof(MumbleInfo)); mumbleinfo = (MumbleInfo *)-1; } 
    if(mumblelink >= 0) { close(mumblelink); mumblelink = -1; }
#endif
}

static inline vec mumblevec(const vec &v, bool pos = false)
{
    // change from X left, Z up, Y forward to X right, Y up, Z forward
    // 8 cube units = 1 meter
    vec m(-v.x, v.z, v.y);
    if(pos) m.div(8);
    return m;
}

void updatemumble()
{
#ifdef VALID_MUMBLELINK
    if(!VALID_MUMBLELINK) return;

    static int timestamp = 0;

    mumbleinfo->version = 1;
    mumbleinfo->timestamp = ++timestamp;

    mumbleinfo->pos = mumblevec(player->o, true);
    mumbleinfo->front = mumblevec(vec(RAD*player->yaw, RAD*player->pitch));
    mumbleinfo->top = mumblevec(vec(RAD*player->yaw, RAD*(player->pitch+90)));
#endif
}

