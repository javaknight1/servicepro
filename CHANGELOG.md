# Changelog

All notable changes to ServicePro will be documented in this file.
## [0.2.0] - 2026-01-24

### Release Features

| Type | Description | Commit |
|------|-------------|--------|
| ✨ Features | Created endpoint to CRUD a user's profile image | [`b157f50`](https://github.com/javaknight1/servicepro/commit/b157f507c1ee116cc61ba7dd6107ba2a0072d354) |
| ✨ Features | Created frontend components to create profile pic for users | [`2066c0f`](https://github.com/javaknight1/servicepro/commit/2066c0f9fcbb6861ea0e1c602de58002bc2732de) |
| 🎯 Minor Changes | Created new version endpoint to return current running version | [`e1965d5`](https://github.com/javaknight1/servicepro/commit/e1965d5e95fc7bfe0ea02e1479f9b3c770b07e0b) |

### Dev Features

| Type | Description | Commit |
|------|-------------|--------|
| ♻️ Refactoring | Removed unused gitops | [`b681a1f`](https://github.com/javaknight1/servicepro/commit/b681a1ff61a331463589ce2c31cdd41f42f96567) |
| ♻️ Refactoring | Centralized config and migrated any client services to factory design | [`e7d827e`](https://github.com/javaknight1/servicepro/commit/e7d827e886cee416863bd1856f7176729870da6a) |
| ♻️ Refactoring | Removed code not being used in backend and recorded into TODO | [`f3ea4d8`](https://github.com/javaknight1/servicepro/commit/f3ea4d82524852b70f1fcd2e10855885efd2ba75) |
| ♻️ Refactoring | Supporting only S3 compatible storage options | [`bc992f6`](https://github.com/javaknight1/servicepro/commit/bc992f67a48b00eb17371f06a87862d4e5744b8e) |
| ✅ Tests | Fixed docker compose and Makefile to work correctly | [`6051f9f`](https://github.com/javaknight1/servicepro/commit/6051f9f863988008215264b51796a9bc5448ae8c) |
| ✅ Tests | Fixed many of the broken unit tests | [`8e1ba2b`](https://github.com/javaknight1/servicepro/commit/8e1ba2b8a81a0aa0b7d3cb4f2402711a7271c2b3) |
| ✅ Tests | Fixed some unit tests | [`b7d56ad`](https://github.com/javaknight1/servicepro/commit/b7d56ad39077e86b98936dbdf4b24177e1c2f7dd) |
| ✅ Tests | Fixed all unit tests for frontend | [`9f25cd8`](https://github.com/javaknight1/servicepro/commit/9f25cd86e420ee7101908e478d1a20025ca94034) |
| ✅ Tests | Fixed breaking unit tests for backend and frontend | [`a1841c7`](https://github.com/javaknight1/servicepro/commit/a1841c717032f21a69ce64e7f10efc26d4dc3b58) |
| ✅ Tests | Streamlined a lot of our scripts | [`a4d03a5`](https://github.com/javaknight1/servicepro/commit/a4d03a56367f42748441fe0665855f8663a3dd94) |
| ✅ Tests | Fixed frontend not being able to run | [`488e1d1`](https://github.com/javaknight1/servicepro/commit/488e1d177e9d622cfddf0cc445cf440c444b83e3) |
| ✅ Tests | Removed test docker compose | [`f91ee65`](https://github.com/javaknight1/servicepro/commit/f91ee652d9ec55d53f40a604ee7603135bb5d958) |
| ✅ Tests | Fixed unit tests with new profile pic handler | [`93ca0bc`](https://github.com/javaknight1/servicepro/commit/93ca0bca7e99b4049cebf43a871c1383359bcb8f) |
| 📚 Documentation | Streamlined and cleaned up all the docs | [`ba6de95`](https://github.com/javaknight1/servicepro/commit/ba6de95028a9207439304b1628fb31d9f3ca7336) |
| 🔧 Miscellaneous | Updated .gitignore for removing terraform, helm, and gitops | [`9240607`](https://github.com/javaknight1/servicepro/commit/9240607051d8022df10cf32bf722685e96d74224) |
| 🔧 Miscellaneous | Removed useless commands in Makefile | [`b766aec`](https://github.com/javaknight1/servicepro/commit/b766aeca5834f7acdd5c5229d4999faf1cbd6564) |
| 🔧 Miscellaneous | Cleaned up change log formatting | [`9a0aff4`](https://github.com/javaknight1/servicepro/commit/9a0aff4c79b5f4adc4dbf41acd24e6fd0f78f2bb) |
| 🚀 CI/CD | Update linting job | [`5997ffe`](https://github.com/javaknight1/servicepro/commit/5997ffe19c81b11e78e075123cb16862e477eea3) |
| 🚀 CI/CD | Ignore linting for some files | [`152a536`](https://github.com/javaknight1/servicepro/commit/152a536c473063df5accf0e2d48150bc71f586e8) |
| 🚀 CI/CD | Streamline the Makefile | [`e70e4fa`](https://github.com/javaknight1/servicepro/commit/e70e4fac1098d8cca53f7489ffa36a3a7e42851c) |
| 🚀 CI/CD | Fixed some linting issues from our CI job | [`d021036`](https://github.com/javaknight1/servicepro/commit/d0210360d42862ec3678fd8a81907128123df95c) |
| 🚀 CI/CD | Clean up all our linting | [`8f9aaed`](https://github.com/javaknight1/servicepro/commit/8f9aaed32e7259ff5c56f972d304259d5c54eae6) |
| 🚀 CI/CD | Cleaned up all our ci jobs | [`f5f02d1`](https://github.com/javaknight1/servicepro/commit/f5f02d1eccf9202173eada36e43006aa98156df7) |
| 🚀 CI/CD | Fixed failing remote unit tests for frontend | [`d6fac12`](https://github.com/javaknight1/servicepro/commit/d6fac126450305edc39b2092ebe5544a1d57b005) |
| 🚀 CI/CD | Pretty linting files | [`9de1414`](https://github.com/javaknight1/servicepro/commit/9de14142670af34428ed816ad464d8b7d80bf206) |
| 🚀 CI/CD | Fixed linting errors | [`3a6489f`](https://github.com/javaknight1/servicepro/commit/3a6489fec8f922193798e73e9b532dee32667371) |
| 🚀 CI/CD | Updated output of change log for new releases | [`f189672`](https://github.com/javaknight1/servicepro/commit/f189672de35e80e6d11f9425853b50ed8a9c6b79) |
| 🚀 CI/CD | Output major and minor commits | [`f2da4fc`](https://github.com/javaknight1/servicepro/commit/f2da4fcd43acdd37221b1af8d780fc8af08cfe65) |
| 🚀 CI/CD | Fixed change log generation commit link | [`7d2b84d`](https://github.com/javaknight1/servicepro/commit/7d2b84d9090abc2f2a542c92e62e89bc35dde15c) |
| 🚀 CI/CD | Fixed change log white space generated | [`671c64e`](https://github.com/javaknight1/servicepro/commit/671c64e89f4dc594df2afa1f5b63374bc9cd51ed) |
## [0.1.0] - 2026-01-23

### Dev Features

| Type | Description | Commit |
|------|-------------|--------|
| 🔧 Miscellaneous | Removed terraform and helm chart since we are moving to a different infra | [`545235c`](https://github.com/javaknight1/servicepro/commit/545235ce4322f943e241ffa9c4656f9a4e21c357) |
| 🔧 Miscellaneous | Added check for valid commit messages | [`dceafa5`](https://github.com/javaknight1/servicepro/commit/dceafa55f27f64e818ba7e359508a547c8c91a13) |
| 🔧 Miscellaneous | Add release pipeline | [`c65d83d`](https://github.com/javaknight1/servicepro/commit/c65d83d36f27abb5c05753fe70d37995792f27f3) |
