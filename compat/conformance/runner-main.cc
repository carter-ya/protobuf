// Run each unmodified upstream suite independently. The upstream combined main
// short-circuits after a failing binary/JSON suite, hiding the text results.
#include <cstdint>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <iostream>
#include <memory>
#include <string>
#include "binary_json_conformance_suite.h"
#include "conformance_test.h"
#include "text_format_conformance_suite.h"
#include "google/protobuf/descriptor.pb.h"
#include "compat/qualify/binary/closed.pb.h"
#include "google/protobuf/dynamic_message.h"
#include "google/protobuf/util/field_comparator.h"
#include "google/protobuf/util/message_differencer.h"

namespace {
constexpr uint32_t kMaxFrame = 1 << 20;
uint32_t LittleEndian(const char* p) {
  const auto* bytes = reinterpret_cast<const unsigned char*>(p);
  return uint32_t(bytes[0]) | uint32_t(bytes[1]) << 8 |
         uint32_t(bytes[2]) << 16 | uint32_t(bytes[3]) << 24;
}

// Private test-only protocol: length-prefixed requests, single-byte responses.
// Op 0 parses one payload; op 1 compares two parsed messages (uint32 left size,
// left wire, right wire). It uses the pinned C++ parser and semantic comparator,
// independent of the SDK and of the Go dynamic closed-enum implementation.
int WireOracle(int argc, char* argv[]) {
  if (argc != 4) return 2;
  (void)qualification::closed::Record::descriptor();
  google::protobuf::DescriptorPool pool(
      google::protobuf::DescriptorPool::generated_pool());
  std::ifstream schema_file(argv[3], std::ios::binary);
  google::protobuf::FileDescriptorSet schema;
  if (!schema_file || !schema.ParseFromIstream(&schema_file)) return 2;
  for (const auto& file : schema.file()) {
    if (pool.FindFileByName(file.name()) == nullptr && pool.BuildFile(file) == nullptr)
      return 2;
  }
  const auto* descriptor = pool.FindMessageTypeByName(argv[2]);
  if (descriptor == nullptr) return 2;
  google::protobuf::DynamicMessageFactory factory(&pool);
  factory.SetDelegateToGeneratedFactory(true);
  const auto* prototype = factory.GetPrototype(descriptor);
  if (prototype == nullptr) return 2;
  google::protobuf::util::DefaultFieldComparator fields;
  fields.set_treat_nan_as_equal(true);
  google::protobuf::util::MessageDifferencer differencer;
  differencer.set_field_comparator(&fields);
  for (;;) {
    char header[4];
    std::cin.read(header, sizeof(header));
    if (std::cin.gcount() == 0 && std::cin.eof()) return 0;
    if (std::cin.gcount() != sizeof(header)) return 2;
    const uint32_t size = LittleEndian(header);
    if (size == 0 || size > kMaxFrame) return 2;
    std::string frame(size, '\0');
    if (!std::cin.read(&frame[0], size)) return 2;
    std::unique_ptr<google::protobuf::Message> left(prototype->New());
    bool result = false;
    if (frame[0] == 0) {
      result = left->ParseFromArray(frame.data() + 1, size - 1);
    } else if (frame[0] == 1 && size >= 5) {
      const uint32_t left_size = LittleEndian(frame.data() + 1);
      if (left_size > size - 5) return 2;
      std::unique_ptr<google::protobuf::Message> right(prototype->New());
      result = left->ParseFromArray(frame.data() + 5, left_size) &&
               right->ParseFromArray(frame.data() + 5 + left_size,
                                     size - 5 - left_size) &&
               differencer.Compare(*left, *right);
    } else {
      return 2;
    }
    std::cout.put(result ? 1 : 0);
    std::cout.flush();
    if (!std::cout) return 2;
  }
}
}  // namespace

int main(int argc, char* argv[]) {
  if (argc < 2) return 2;
  if (std::strcmp(argv[1], "wire_oracle") == 0) return WireOracle(argc, argv);
  google::protobuf::BinaryAndJsonConformanceSuite binary_json;
  google::protobuf::TextFormatConformanceTestSuite text;
  google::protobuf::ConformanceTestSuite* suite = nullptr;
  if (std::strcmp(argv[1], "binary_json") == 0) suite = &binary_json;
  if (std::strcmp(argv[1], "text") == 0) suite = &text;
  if (suite == nullptr) return 2;
  return google::protobuf::RunConformanceTests(argc - 1, argv + 1, {suite});
}
