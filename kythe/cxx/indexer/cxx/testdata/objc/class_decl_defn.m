// Checks that Objective-C classes are declared and defined.
// todo(salguarnieri) Have the implementation "completes/uniquely" the
// interface.

//- @Box defines/binding BoxIface
//- BoxIface.node/kind record
//- BoxIface.subkind class
//- BoxIface.complete incomplete
@interface Box
@end

// Doesn't work yet
//#- @Box completes/uniquely BoxIface

//- @Box defines/binding BoxImpl
//- BoxImpl.node/kind record
//- BoxImpl.subkind class
//- BoxImpl.complete definition
@implementation Box
@end

//- BoxIface named BoxName
//- BoxImpl named BoxName

int main(int argc, char **argv) {
  return 0;
}

