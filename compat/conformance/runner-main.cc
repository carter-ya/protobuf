// Run each unmodified upstream suite independently. The upstream combined main
// short-circuits after a failing binary/JSON suite, hiding the text results.
#include <cstring>
#include <cstdio>
#include "binary_json_conformance_suite.h"
#include "conformance_test.h"
#include "text_format_conformance_suite.h"
int main(int argc, char* argv[]) {
  if (argc < 2) return 2;
  google::protobuf::BinaryAndJsonConformanceSuite binary_json;
  google::protobuf::TextFormatConformanceTestSuite text;
  google::protobuf::ConformanceTestSuite* suite = nullptr;
  if (std::strcmp(argv[1], "binary_json") == 0) suite = &binary_json;
  if (std::strcmp(argv[1], "text") == 0) suite = &text;
  if (suite == nullptr) return 2;
  return google::protobuf::RunConformanceTests(argc - 1, argv + 1, {suite});
}
