// this file defines static map entities ("entity") and dynamic entities (players/monsters, "dynent")
// the gamecode extends these types to add game specific functionality

// ET_*: the only static entity types dictated by the engine... rest are gamecode dependent

enum { ET_EMPTY=0, ET_LIGHT, ET_MAPMODEL, ET_PLAYERSTART, ET_ENVMAP, ET_PARTICLES, ET_SOUND, ET_SPOTLIGHT, ET_GAMESPECIFIC };

struct entity                                   // persistent map entity
{
    vec o;                                      // position
    short attr1, attr2, attr3, attr4, attr5;
    uchar type;                                 // type is one of the above
    uchar reserved;
};

struct entitylight
{
    vec color, dir;
    int millis;

    entitylight() : color(1, 1, 1), dir(0, 0, 1), millis(-1) {}
};

enum
{
    EF_NOVIS     = 1<<0,
    EF_NOSHADOW  = 1<<1,
    EF_NOCOLLIDE = 1<<2,
    EF_ANIM      = 1<<3,
    EF_OCTA      = 1<<4,
    EF_RENDER    = 1<<5,
    EF_SOUND     = 1<<6,
    EF_SPAWNED   = 1<<7,
    EF_NOPICKUP  = 1<<8
};

struct extentity : entity                       // part of the entity that doesn't get saved to disk
{
    int flags;  // the only dynamic state of a map entity
    entitylight light;
    extentity *attached;

    extentity() : flags(0), attached(NULL) {}

    bool spawned() const { return (flags&EF_SPAWNED) != 0; }
    void setspawned(bool val) { if(val) flags |= EF_SPAWNED; else flags &= ~EF_SPAWNED; }
    void setspawned() { flags |= EF_SPAWNED; }
    void clearspawned() { flags &= ~EF_SPAWNED; }

    bool nopickup() const { return (flags&EF_NOPICKUP) != 0; }
    void setnopickup(bool val) { if(val) flags |= EF_NOPICKUP; else flags &= ~EF_NOPICKUP; }
    void setnopickup() { flags |= EF_NOPICKUP; }
    void clearnopickup() { flags &= ~EF_NOPICKUP; }
};

#define MAXENTS 10000

//extern vector<extentity *> ents;                // map entities

enum { CS_ALIVE = 0, CS_DEAD, CS_SPAWNING, CS_LAGGED, CS_EDITING, CS_SPECTATOR };

enum { PHYS_FLOAT = 0, PHYS_FALL, PHYS_SLIDE, PHYS_SLOPE, PHYS_FLOOR, PHYS_STEP_UP, PHYS_STEP_DOWN, PHYS_BOUNCE };

enum { ENT_PLAYER = 0, ENT_AI, ENT_INANIMATE, ENT_CAMERA, ENT_BOUNCE };

enum { COLLIDE_NONE = 0, COLLIDE_ELLIPSE, COLLIDE_OBB, COLLIDE_ELLIPSE_PRECISE };

struct physent                                  // base entity type, can be affected by physics
{
    vec o, vel, falling;                        // origin, velocity
    vec deltapos, newpos;                       // movement interpolation
    float yaw, pitch, roll;
    float maxspeed;                             // cubes per second, 100 for player
    float radius, eyeheight, aboveeye;          // bounding box size
    float xradius, yradius, zmargin;
    vec floor;                                  // the normal of floor the dynent is on

    ushort timeinair;
    uchar inwater;
    bool jumping;
    schar move, strafe;

    uchar physstate;                            // one of PHYS_* above
    uchar state, editstate;                     // one of CS_* above
    uchar type;                                 // one of ENT_* above
    uchar collidetype;                          // one of COLLIDE_* above           

    bool blocked;                               // used by physics to signal ai

    physent() : o(0, 0, 0), deltapos(0, 0, 0), newpos(0, 0, 0), yaw(0), pitch(0), roll(0), maxspeed(100), 
               radius(4.1f), eyeheight(14), aboveeye(1), xradius(4.1f), yradius(4.1f), zmargin(0),
               state(CS_ALIVE), editstate(CS_ALIVE), type(ENT_PLAYER),
               collidetype(COLLIDE_ELLIPSE),
               blocked(false)
               { reset(); }
              
    void resetinterp()
    {
        newpos = o;
        deltapos = vec(0, 0, 0);
    }

    void reset()
    {
        inwater = 0;
        timeinair = 0;
        jumping = false;
        strafe = move = 0;
        physstate = PHYS_FALL;
        vel = falling = vec(0, 0, 0);
        floor = vec(0, 0, 1);
    }

    vec feetpos(float offset = 0) const { return vec(o).add(vec(0, 0, offset - eyeheight)); }
    vec headpos(float offset = 0) const { return vec(o).add(vec(0, 0, offset)); }

    bool maymove() const { return timeinair || physstate < PHYS_FLOOR || vel.squaredlen() > 1e-4f || deltapos.squaredlen() > 1e-4f; } 
};

enum
{
    ANIM_DEAD = 0, ANIM_DYING, ANIM_IDLE,
    ANIM_FORWARD, ANIM_BACKWARD, ANIM_LEFT, ANIM_RIGHT,
    ANIM_HOLD1, ANIM_HOLD2, ANIM_HOLD3, ANIM_HOLD4, ANIM_HOLD5, ANIM_HOLD6, ANIM_HOLD7,
    ANIM_ATTACK1, ANIM_ATTACK2, ANIM_ATTACK3, ANIM_ATTACK4, ANIM_ATTACK5, ANIM_ATTACK6, ANIM_ATTACK7,
    ANIM_PAIN,
    ANIM_JUMP, ANIM_SINK, ANIM_SWIM,
    ANIM_EDIT, ANIM_LAG, ANIM_TAUNT, ANIM_WIN, ANIM_LOSE,
    ANIM_GUN_IDLE, ANIM_GUN_SHOOT,
    ANIM_VWEP_IDLE, ANIM_VWEP_SHOOT, ANIM_SHIELD, ANIM_POWERUP,
    ANIM_MAPMODEL, ANIM_TRIGGER,
    NUMANIMS
};

static const char * const animnames[] =
{
    "dead", "dying", "idle",
    "forward", "backward", "left", "right",
    "hold 1", "hold 2", "hold 3", "hold 4", "hold 5", "hold 6", "hold 7",
    "attack 1", "attack 2", "attack 3", "attack 4", "attack 5", "attack 6", "attack 7",
    "pain",
    "jump", "sink", "swim",
    "edit", "lag", "taunt", "win", "lose",
    "gun idle", "gun shoot",
    "vwep idle", "vwep shoot", "shield", "powerup",
    "mapmodel", "trigger"
};

#define ANIM_ALL         0x7F
#define ANIM_INDEX       0x7F
#define ANIM_LOOP        (1<<7)
#define ANIM_START       (1<<8)
#define ANIM_END         (1<<9)
#define ANIM_REVERSE     (1<<10)
#define ANIM_CLAMP       (ANIM_START|ANIM_END)
#define ANIM_DIR         0x780
#define ANIM_SECONDARY   11
#define ANIM_NOSKIN      (1<<22)
#define ANIM_SETTIME     (1<<23)
#define ANIM_FULLBRIGHT  (1<<24)
#define ANIM_REUSE       (1<<25)
#define ANIM_NORENDER    (1<<26)
#define ANIM_RAGDOLL     (1<<27)
#define ANIM_SETSPEED    (1<<28)
#define ANIM_NOPITCH     (1<<29)
#define ANIM_GHOST       (1<<30)
#define ANIM_FLAGS       (0x1FF<<22)

struct animinfo // description of a character's animation
{
    int anim, frame, range, basetime;
    float speed;
    uint varseed;

    animinfo() : anim(0), frame(0), range(0), basetime(0), speed(100.0f), varseed(0) { }

    bool operator==(const animinfo &o) const { return frame==o.frame && range==o.range && (anim&(ANIM_SETTIME|ANIM_DIR))==(o.anim&(ANIM_SETTIME|ANIM_DIR)) && (anim&ANIM_SETTIME || basetime==o.basetime) && speed==o.speed; }
    bool operator!=(const animinfo &o) const { return frame!=o.frame || range!=o.range || (anim&(ANIM_SETTIME|ANIM_DIR))!=(o.anim&(ANIM_SETTIME|ANIM_DIR)) || (!(anim&ANIM_SETTIME) && basetime!=o.basetime) || speed!=o.speed; }
};

struct animinterpinfo // used for animation blending of animated characters
{
    animinfo prev, cur;
    int lastswitch;
    void *lastmodel;

    animinterpinfo() : lastswitch(-1), lastmodel(NULL) {}

    void reset() { lastswitch = -1; }
};

#define MAXANIMPARTS 3

struct occludequery;
struct ragdolldata;

struct dynent : physent                         // animated characters, or characters that can receive input
{
    bool k_left, k_right, k_up, k_down;         // see input code

    entitylight light;
    animinterpinfo animinterp[MAXANIMPARTS];
    ragdolldata *ragdoll;
    occludequery *query;
    int lastrendered;
    uchar occluded;

    dynent() : ragdoll(NULL), query(NULL), lastrendered(0), occluded(0)
    { 
        reset(); 
    }

    ~dynent()
    {
#ifndef STANDALONE
        extern void cleanragdoll(dynent *d);
        if(ragdoll) cleanragdoll(this);
#endif
    }
               
    void stopmoving()
    {
        k_left = k_right = k_up = k_down = jumping = false;
        move = strafe = 0;
    }
        
    void reset()
    {
        physent::reset();
        stopmoving();
        loopi(MAXANIMPARTS) animinterp[i].reset();
    }

    vec abovehead() { return vec(o).add(vec(0, 0, aboveeye+4)); }
};


