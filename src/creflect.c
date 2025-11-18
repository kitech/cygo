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
P00_SPRINT_DEFINE(schar, "%hhd");
P00_SPRINT_DEFINE(uchar, "%hhu", "%#hhX", "%#hho");
P00_SPRINT_DEFINE(short, "%hd");
P00_SPRINT_DEFINE(ushort, "%hu", "%#hX", "%#ho");
P00_SPRINT_DEFINE(unsigned, "%u", "%#X", "%#o");
P00_SPRINT_DEFINE(ulong, "%lu", "%#lX", "%#lo");
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
        tystr = ctypenstr_other;
        break;
     case ctypeid_bool:
        tystr = ctypenstr_bool;
        break;
     case ctypeid_char:
        tystr = ctypenstr_char;
        fmtstr = "%c";
        break;
     case ctypeid_uchar:
        tystr = ctypenstr_uchar;
        break;
     case ctypeid_short:
        tystr = ctypenstr_short;
        break;
     case ctypeid_ushort:
        tystr = ctypenstr_ushort;
        break;
     case ctypeid_int:
        tystr = ctypenstr_int;
        break;
     case ctypeid_uint:
        tystr = ctypenstr_uint;
        break;
     case ctypeid_long:
        tystr = ctypenstr_long;
        fmtstr = "%ld";
        break;
     case ctypeid_ulong:
        tystr = ctypenstr_ulong;
        fmtstr = "%lu";
        break;
     case ctypeid_longlong:
        tystr = ctypenstr_longlong;
        fmtstr = "%lld";
        break;
     case ctypeid_ulonglong:
        tystr = ctypenstr_ulonglong;
        fmtstr = "%llu";
        break;
     case ctypeid_float:
        tystr = ctypenstr_float;
        fmtstr = "%f";
        break;
     case ctypeid_double:
        tystr = ctypenstr_double;
        break;
     case ctypeid_longdouble:
        tystr = ctypenstr_longdouble;
        break;
     case ctypeid_charptr:
        tystr = ctypenstr_charptr;
        fmtstr = "%s";
        break;
     case ctypeid_voidptr:
        tystr = ctypenstr_voidptr;
        fmtstr = "%p";
        break;
     case ctypeid_intptr:
        tystr = ctypenstr_intptr;
        break;
    case ctypeid_func_void:
        tystr = "void(*)()";
        fmtstr = "%p";
        break;
    }
    #undef caseid2str
    return tystr_or_fmtstr==1 ? fmtstr : tystr ;
}

const char* ctypeid_tostr(int tyid) {
    return ctypeid_toany_impl(tyid, 0);
}
const char* ctypeid_tofmt(int tyid) {
    return ctypeid_toany_impl(tyid, 1);
}
int ctypeid_is_anyint(int tyid) {
    switch (tyid) {
        case ctypeid_int: case ctypeid_uint:
        case ctypeid_long: case ctypeid_ulong:
        case ctypeid_short: case ctypeid_ushort:
        return 1;
    }
    return 0;
}
int ctypeid_is_anyreal(int tyid) {
    return tyid==ctypeid_float || tyid==ctypeid_double||tyid==ctypeid_longdouble;
}
int ctypeid_is_anyptr(int tyid) {
    switch (tyid) {
        case ctypeid_intptr: case ctypeid_charptr:
        case ctypeid_charptrptr:
        case ctypeid_voidptr:
        return 1;
    }
    return 0;
}
int ctypeid_is_anyfun(int tyid) {
    switch (tyid) {
        case ctypeid_func_int: case ctypeid_func_int32:
        case ctypeid_func_int64: case ctypeid_func_usize:
        case ctypeid_func_charptr: case ctypeid_charptrptr:
        case ctypeid_func_double: case ctypeid_func_float:
        case ctypeid_func_void:
        return 1;
    }
    return 0;
}
