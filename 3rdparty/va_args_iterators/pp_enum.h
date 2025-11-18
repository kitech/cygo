/**
 * @file
 *
 * Tagged Enum - automated generation of string tags associated with enum members
 *
 * @author Cormac Cannon (cormacc-public@gmail.com)
 *
 * These macros are used to generate enums and associated tags without repetition, e.g.
 *
 * 1. Define the unique names in a new macro
 * #define MyEnumIds NAME_1, NAME_2, NAME_3
 *
 * 2. Create the enum (In the .h, typically)
 * TAGGED_ENUM(MyEnum);
 *    ... or, for a typedeffed enum ...
 * TAGGED_ENUM_TYPE(MyEnum);
 *
 * 3. Create the tags (In the .c, typically)
 * ENUM_DESCRIBE(MyEnum);
 *
 * The defined enum / enum type will include a `<Name>_COUNT` member.
 * The header will declare the function ~char const * MyEnum_asCString(int)~
 * The source file will define an array of strings using the defined MyEnumIds and
 * the implementation of ~char const * MyEnum_asCString(int)~
 *
 * N.B. Tag generation uses VA_ARGS counting and iteration macros defined in "pp_iter.h"
 * See that file for a link to a generator script if higher limit is required.
 * That'll take precedence over this include, as they use the same guard definition -- i.e
 * #include "pp_iter512.h"
 * #include "enum.h"
 */

#ifndef Enum_H
#  define Enum_H

#  ifdef  __cplusplus
extern "C" {
#  endif

// ----------------------------------------------------------------------------
// Includes
// ----------------------------------------------------------------------------
#include "pp_iter.h"

// ----------------------------------------------------------------------------
// Macros
// ----------------------------------------------------------------------------

/**
 * Override this with another string literal
 * Alternatively, override with an external string constant to save a few bytes per tagged enum
 *
 * e.g. in 'my_tagged_enum.h'
 *
 *      extern char const GLOBAL_ENUM_UNDEFINED_TAG[];
 *      #define ENUM_UNDEFINED_TAG GLOBAL_ENUM_UNDEFINED_TAG
 *      #include "enum.h"
 *
 * and in 'my_tagged_enum.c'
 *
 *      char const GLOBAL_ENUM_UNDEFINED_TAG[] = "N.A."; //or, in the immortal words of Seamus Ennis, 'Whateveryouplays, sur'
 */
#ifndef ENUM_UNDEFINED_TAG
#define ENUM_UNDEFINED_TAG "UNDEFINED"
#endif

#define _ENUM_MEMBER_(NAME) NAME,
#define _ENUM_TAG_(NAME) #NAME,

#define _ENUM_DESCRIPTOR_DECLARATION_(NAME)           \
  char const * NAME##_asCString(int id)

#define _ENUM_DESCRIPTOR_DEFINITION_(NAME)                                    \
  _ENUM_DESCRIPTOR_DECLARATION_(NAME) { return id < NAME##_COUNT ? NAME##_TAGS[id] : ENUM_UNDEFINED_TAG; }

#define ENUM(NAME) enum NAME { NAME##Tags , NAME##_COUNT }
#define ENUM_TYPE(NAME) typedef enum { NAME##Tags , NAME##_COUNT } NAME;

#define TAGGED_ENUM(NAME)                       \
  ENUM(NAME);                                   \
  _ENUM_DESCRIPTOR_DECLARATION_(NAME)

#define TAGGED_ENUM_TYPE(NAME)                       \
  ENUM_TYPE(NAME);                                   \
  _ENUM_DESCRIPTOR_DECLARATION_(NAME)

// Indirection necessary to expand NAME##Tags
#define _ENUM_TAGS_AUTO_(IDS) PP_EACH(_ENUM_TAG_, IDS)
#define ENUM_TAGS_AUTO(NAME)                         \
  static char const * NAME##_TAGS[] = {         \
    _ENUM_TAGS_AUTO_(NAME##Tags)                      \
  }

//Explicit array size used here to keep us honest -- i.e. ensure
// number of tags provided matches enum member count....
#define ENUM_TAGS_EXPLICIT(NAME, ...)               \
  static char const * NAME##_TAGS[NAME##_COUNT] = { \
    __VA_ARGS__                                     \
  }

#define ENUM_DESCRIBE(NAME)                          \
  ENUM_TAGS_AUTO(NAME);                              \
  _ENUM_DESCRIPTOR_DEFINITION_(NAME)

#define ENUM_DESCRIBE_EXPLICIT(NAME, ...)    \
  ENUM_TAGS_EXPLICIT(NAME, __VA_ARGS__);     \
  _ENUM_DESCRIPTOR_DEFINITION_(NAME)


#  ifdef  __cplusplus
}
#  endif

#endif  /* Enum_H */
