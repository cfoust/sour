#include "game.h"

namespace entities
{
    using namespace game;

    vector<extentity *> ents;

    vector<extentity *> &getents() { return ents; }

    extentity *newentity() { return new fpsentity(); }
    void deleteentity(extentity *e) { delete (fpsentity *)e; }

    void clearents()
    {
        while(ents.length()) deleteentity(ents.pop());
    }

    enum
    {
        TRIG_COLLIDE    = 1<<0,
        TRIG_TOGGLE     = 1<<1,
        TRIG_ONCE       = 0<<2,
        TRIG_MANY       = 1<<2,
        TRIG_DISAPPEAR  = 1<<3,
        TRIG_AUTO_RESET = 1<<4,
        TRIG_RUMBLE     = 1<<5,
        TRIG_LOCKED     = 1<<6,
        TRIG_ENDSP      = 1<<7
    };

    static const int NUMTRIGGERTYPES = 32;

    static const int triggertypes[NUMTRIGGERTYPES] =
    {
        -1,
        TRIG_ONCE,                    // 1
        TRIG_RUMBLE,                  // 2
        TRIG_TOGGLE,                  // 3
        TRIG_TOGGLE | TRIG_RUMBLE,    // 4
        TRIG_MANY,                    // 5
        TRIG_MANY | TRIG_RUMBLE,      // 6
        TRIG_MANY | TRIG_TOGGLE,      // 7
        TRIG_MANY | TRIG_TOGGLE | TRIG_RUMBLE,    // 8
        TRIG_COLLIDE | TRIG_TOGGLE | TRIG_RUMBLE, // 9
        TRIG_COLLIDE | TRIG_TOGGLE | TRIG_AUTO_RESET | TRIG_RUMBLE, // 10
        TRIG_COLLIDE | TRIG_TOGGLE | TRIG_LOCKED | TRIG_RUMBLE,     // 11
        TRIG_DISAPPEAR,               // 12
        TRIG_DISAPPEAR | TRIG_RUMBLE, // 13
        TRIG_DISAPPEAR | TRIG_COLLIDE | TRIG_LOCKED, // 14
        -1 /* reserved 15 */,
        -1 /* reserved 16 */,
        -1 /* reserved 17 */,
        -1 /* reserved 18 */,
        -1 /* reserved 19 */,
        -1 /* reserved 20 */,
        -1 /* reserved 21 */,
        -1 /* reserved 22 */,
        -1 /* reserved 23 */,
        -1 /* reserved 24 */,
        -1 /* reserved 25 */,
        -1 /* reserved 26 */,
        -1 /* reserved 27 */,
        -1 /* reserved 28 */,
        TRIG_DISAPPEAR | TRIG_RUMBLE | TRIG_ENDSP, // 29
        -1 /* reserved 30 */,
        -1 /* reserved 31 */,
    };

    #define validtrigger(type) (triggertypes[(type) & (NUMTRIGGERTYPES-1)]>=0)
    #define checktriggertype(type, flag) (triggertypes[(type) & (NUMTRIGGERTYPES-1)] & (flag))

    static inline void cleartriggerflags(extentity &e)
    {
        e.flags &= ~(EF_ANIM | EF_NOVIS | EF_NOSHADOW | EF_NOCOLLIDE);
    }

    static inline void setuptriggerflags(fpsentity &e)
    {
        cleartriggerflags(e);
        e.flags |= EF_ANIM;
        if(checktriggertype(e.attr3, TRIG_COLLIDE|TRIG_DISAPPEAR)) e.flags |= EF_NOSHADOW;
        if(!checktriggertype(e.attr3, TRIG_COLLIDE)) e.flags |= EF_NOCOLLIDE;
        switch(e.triggerstate)
        {
            case TRIGGERING:
                if(checktriggertype(e.attr3, TRIG_COLLIDE) && lastmillis-e.lasttrigger >= 500) e.flags |= EF_NOCOLLIDE;
                break;
            case TRIGGERED:
                if(checktriggertype(e.attr3, TRIG_COLLIDE)) e.flags |= EF_NOCOLLIDE;
                break;
            case TRIGGER_DISAPPEARED:
                e.flags |= EF_NOVIS | EF_NOCOLLIDE;
                break;
        }
    }

    void editent(int i, bool local)
    {
        extentity &e = *ents[i];
        if(e.type == ET_MAPMODEL && validtrigger(e.attr3))
        {
            fpsentity &f = (fpsentity &)e;
            f.triggerstate = TRIGGER_RESET;
            f.lasttrigger = 0;
            setuptriggerflags(f);
        }
        else cleartriggerflags(e);
        if(local) addmsg(N_EDITENT, "rii3ii5", i, (int)(e.o.x*DMF), (int)(e.o.y*DMF), (int)(e.o.z*DMF), e.type, e.attr1, e.attr2, e.attr3, e.attr4, e.attr5);
    }
}

