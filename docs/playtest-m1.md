# M1 플레이테스트 체크리스트

> 2026-10-11 작성 · M1 통과 기준과 [roadmap.md](roadmap.md)의 "M1 끝" 🎮 결정 항목을 직접 플레이로 판정하기 위한 문서
> 판정 결과는 아래 "결과" 칸에 적고, 확정한 값은 [design.md](design.md)에 ✅로 옮긴다.

## 1. 실행 방법

| 목적 | 명령 |
|---|---|
| 한 판 (1~2분 분량 탑, 물 있음) | `go run ./cmd/uplunge -reload` |
| 같은 탑 다시 | `go run ./cmd/uplunge -seed <로그의 시드> -reload` |
| 비율 실험 (적 간격 고정 통로) | `go run ./cmd/uplunge -lab -reload` |
| 블록만 시험 (물 없음) | `go run ./cmd/uplunge -chunk Blocks -reload` |
| 적만 시험 (물 없음) | `go run ./cmd/uplunge -chunk Enemies -reload` |
| 플레이 녹화 (공유용) | `go run ./cmd/uplunge -record out/<이름>.rpl` (`-reload`와 같이 못 씀) |

- 조작: ← → 또는 A D 이동, Z 또는 Space 점프/발사, R 재시작 (판이 끝나거나 클리어한 뒤).
- `-reload`로 실행하면 `data/tuning.json`을 저장할 때마다 0.5초 안에 반영됨. 적, 탑 구성, 실험 통로는 다음 판(R)부터.
- 화면 왼쪽 위: 연료, HP, 물까지 거리, **비율 = 탄창 높이 / 평균 적 간격**.
- 탑 꼭대기에 닿으면 클리어 시간과 남은 HP가 나옴.

## 2. 통과 기준

- [ ] 1~2분 플레이가 "한 판 더"를 부른다.
- [ ] "탄창 높이 ÷ 적 간격" 비율의 적정 범위를 찾는다.

## 3. 비율 찾기 절차

1. `go run ./cmd/uplunge -lab -reload`
2. `data/tuning.json`의 `lab.spacing`을 바꾸고 R. 탄창 높이(지금 약 134px)를 기준으로 아래 값을 차례로 시험.

| `lab.spacing` | 비율 (탄창 134px 기준) | 예상 |
|---|---|---|
| 64 | 약 2.1 | 적만 밟아도 쉽게 계속 오름 |
| 96 | 약 1.4 | 밟기 연쇄가 됨 (지금 기본값) |
| 128 | 약 1.05 | 경계: 정확히 쏘고 밟아야 이어짐 |
| 160 | 약 0.84 | 발판이 필요 |
| 192 | 약 0.7 | 적은 보조, 발판 위주 |

3. 각 간격에서 볼 것: 밟기 연쇄가 재미있게 이어지는가, 너무 쉬운가, 물에 쫓겨 급한가.
4. 결과로 "구역 1은 비율 ○○, 마지막 구역은 ○○" 같은 범위를 정함. 이 범위는 M2 조각 생성기와 구역 난이도의 기준이 됨 (design.md 4장).

## 4. 판정 항목 (🎮)

각 항목은 `data/tuning.json`에서 바로 바꿔 볼 수 있음. 현재 값은 모두 가설(🧪).

| 항목 | 현재 값 | 바꿀 곳 | 결과 |
|---|---|---|---|
| 추진력 (발사 한 발의 힘, 간격, 탄창) | 240px/s, 0.1초, 8발 | `gun.thrust`, `gun.fireInterval`, `gun.magazine` | |
| 점프 높이와 가변 점프 | 210px/s를 최대 0.2초 | `player.jumpSpeed`, `player.jumpHoldTime` | |
| 꼭대기 중력 절반 | 속도 80 미만에서 × 0.5 | `player.apexGravThreshold`, `player.apexGravMult` | |
| 좌우 이동과 미끄러짐 | 180px/s, 가속·감속 2000, 공중 × 0.65 | `player.runSpeed`, `player.runAccel`, `player.airAccelMult` | |
| 밟기 바운스 | 280px/s를 0.2초 | `player.stompSpeed`, `player.stompHoldTime` | |
| 넉백 (Downwell 측정값) | 좌우 180 × 각도, 위 190 | `player.knockbackX`, `player.knockbackY` | |
| 무적 시간 | 1.5초 | `player.invulnTime` | |
| 드릴 튕김 | 240px/s 한 번 | `player.drillBounce` | |
| 물 속도 | 30px/s (화면 밖이면 최대 3배) | `water.speed`, `water.maxMult` | |
| 물 안전망 튕김 | 400px/s | `water.bounce` | |
| 카메라 위치·미리 보기·추적 | 0.667, 0.2초, 1% | `camera.anchor`, `camera.lookahead`, `camera.remainPerSecond` | |
| 멈춤·흔들림·늘어남 | 0.05초, 0.2초 × 20px, (0.6, 1.4) / (1.6, 0.4) | `feel` | |
| 판정 박스 크기 | 12×20 (Celeste 환산 16×22도 시험) | `player.width`, `player.height` | |
| 기본 키 배치 | 위 조작 표 | 코드 (`internal/input`) | |
| 남길 블록 종류 | 드릴 / 무른 / 총알 전용 모두 있음 | `-chunk Blocks`, 탑의 DrillGate·SoftFloor | |
| 적 속도와 체력 | Floater 30px/s·2발, Spiker 제자리·3발 | `enemies` | |
| 탑 길이 (1~2분 맞는지) | 조각 24개 | `tower.length` | |

## 5. 결과 기록

- 날짜:
- 통과 기준 판정:
- 비율 적정 범위:
- 확정한 수치 (design.md에 ✅로 옮길 것):
- 다음에 고칠 것:
