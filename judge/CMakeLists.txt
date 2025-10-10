cmake_minimum_required(VERSION 3.16)
project(CodeJudgeService)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Find required packages
find_package(PkgConfig REQUIRED)

# Find PostgreSQL
find_package(PostgreSQL REQUIRED)

# Find hiredis
pkg_check_modules(HIREDIS REQUIRED hiredis)

# Find libseccomp
pkg_check_modules(LIBSECCOMP REQUIRED libseccomp)

# Find nlohmann/json
find_package(nlohmann_json REQUIRED)

# Include directories
include_directories(${PostgreSQL_INCLUDE_DIRS})
include_directories(${HIREDIS_INCLUDE_DIRS})
include_directories(${LIBSECCOMP_INCLUDE_DIRS})

# Add executable
add_executable(judge-service-modern
    modern_main.cpp
    sandbox.cpp
)

# Link libraries
target_link_libraries(judge-service-modern
    ${PostgreSQL_LIBRARIES}
    ${HIREDIS_LIBRARIES}
    ${LIBSECCOMP_LIBRARIES}
    nlohmann_json::nlohmann_json
    pthread
)

# Compiler flags
target_compile_options(judge-service-modern PRIVATE
    ${HIREDIS_CFLAGS_OTHER}
    ${LIBSECCOMP_CFLAGS_OTHER}
    -Wall -Wextra -O2 -g
)

# Legacy version disabled - use judge-service-modern only
# add_executable(judge-service-legacy
#     main.cpp
# )
# 
# target_link_libraries(judge-service-legacy
#     ${PostgreSQL_LIBRARIES}
#     ${HIREDIS_LIBRARIES}
#     pthread
# )
# 
# target_compile_options(judge-service-legacy PRIVATE
#     ${HIREDIS_CFLAGS_OTHER}
#     -Wall -Wextra -O2 -g
# )