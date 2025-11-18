#include "cxtypedefs.h"


#define ctypenstr_other "other"
#define ctypenstr_bool "_Bool"
#define ctypenstr_char "char"
#define ctypenstr_uchar "uchar"
#define ctypenstr_short "short"
#define ctypenstr_ushort "ushort"
#define ctypenstr_int "int"
#define ctypenstr_uint "uint"
#define ctypenstr_long "long"
#define ctypenstr_ulong "ulong"
#define ctypenstr_longlong "longlong"
#define ctypenstr_ulonglong "ulonglong"
#define ctypenstr_float "float"
#define ctypenstr_double "double"
#define ctypenstr_longdouble "longdouble"
#define ctypenstr_charptr "charptr"
#define ctypenstr_voidptr "voidptr"
#define ctypenstr_intptr "intptr"

/*
P00_SPRINT_DEFINE(char, "%c");
P00_SPRINT_DEFINE(schar, "%hhd");
P00_SPRINT_DEFINE(uchar, "%hhu", "%#hhX", "%#hho");
P00_SPRINT_DEFINE(short, "%hd");
P00_SPRINT_DEFINE(ushort, "%hu", "%#hX", "%#ho");
P00_SPRINT_DEFINE(int, "%d");
P00_SPRINT_DEFINE(unsigned, "%u", "%#X", "%#o");
P00_SPRINT_DEFINE(long, "%ld");
P00_SPRINT_DEFINE(ulong, "%lu", "%#lX", "%#lo");
P00_SPRINT_DEFINE(llong, "%lld");
P00_SPRINT_DEFINE(ullong, "%llu", "%#llX", "%#llo");
P00_SPRINT_DEFINE(float, "%g", "%a");
P00_SPRINT_DEFINE(double, "%g", "%a");
P00_SPRINT_DEFINE(ldouble, "%Lg", "%La");
 */
const char* ctypeid_toany_impl(int tyid, int tystr_or_fmtstr) {
    char* tystr = ctypenstr_other;
    char *fmtstr = "%d"; 
    #define caseid2str(ty) case ctypeid_##ty: return ctypenstr_##ty
    switch (tyid) {
     case ctypeid_other :
        return ctypenstr_other;
     case ctypeid_bool: 
        return ctypenstr_bool;
     case ctypeid_char: 
        return ctypenstr_char;
     case ctypeid_uchar: 
        return ctypenstr_uchar;
     case ctypeid_short: 
        return ctypenstr_short;
     case ctypeid_ushort: 
        return ctypenstr_ushort;
     case ctypeid_int: 
        return ctypenstr_int;
     case ctypeid_uint: 
        return ctypenstr_uint;
     case ctypeid_long: 
        return ctypenstr_long;
     case ctypeid_ulong: 
        return ctypenstr_ulong;
     case ctypeid_longlong: 
        return ctypenstr_longlong;
     case ctypeid_ulonglong: 
        return ctypenstr_ulonglong;
     case ctypeid_float: 
        return ctypenstr_float;
     case ctypeid_double: 
        return ctypenstr_double;
     case ctypeid_longdouble: 
        return ctypenstr_longdouble;
     case ctypeid_charptr: 
        return ctypenstr_charptr;
     case ctypeid_voidptr: 
        return ctypenstr_voidptr;
     case ctypeid_intptr: 
        return ctypenstr_intptr;
    }
    #undef caseid2str
    return tystr_or_fmtstr==1 ? fmtstr : ctypenstr_other ;
}

const char* ctypeid_tostr(int tyid) {
    return ctypeid_toany_impl(tyid, 0);
}
const char* ctypeid_tofmt(int tyid) {
    return ctypeid_toany_impl(tyid, 1);
}
int ctypeid_is_anyint(int tyid) {
    return 0;
}
int ctypeid_is_anyreal(int tyid) {
    return 0;
}
int ctypeid_is_anyptr(int tyid) {
    return 0;
}
