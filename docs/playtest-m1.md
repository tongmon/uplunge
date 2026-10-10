# M1 플레이테스트 체크리스트

> 2026-10-11 작성 · M1 통과 기준과 [roadmap.md](roadmap.md)의 "M1 끝" 🎮 결정 항목을 직접 플레이로 판정하기 위한 문서
> 판정 결과는 아래 "결과" 칸에 적고, 확정한 값은 [design.md](design.md)에 ✅로 옮긴다.

## 1. 실행 방법

| 목적 | 명령 |
|---|---|
| 한 판 (1~2분 분량 탑, 물 있음) | `go run ./cmd/uplunge -reload` |
| 같은 탑 다시 | `go run ./cmd/uplunge -seed <로그의 시드> -reload` (실험 통로는 `-lab -seed <시드>`) |
| 비율 실험 (적 간격 고정 통로) | `go run ./cmd/uplunge -lab -reload` |
| 블록만 시험 (물 없음) | `go run ./cmd/uplunge -chunk Blocks -reload` |
| 적만 시험 (물 없음) | `go run ./cmd/uplunge -chunk Enemies -reload` |
| 플레이 녹화 (공유용) | `go run ./cmd/uplunge -record out/<이름>.rpl` (`-reload`와 같이 못 씀) |

- 조작: ← → 또는 A D 이동, Z 또는 Space 점프/발사, R 재시작 (탑은 판이 끝나거나 클리어한 뒤, 실험 통로는 언제든).
- `-reload`로 실행하면 `data/tuning.json`을 저장할 때마다 0.5초 안에 반영됨. 적, 탑 구성, 실험 통로는 다음 판(R)부터.
- 화면 왼쪽 위: 연료, HP, 물까지 거리, **비율 = 탄창 높이 / 평균 적 간격**.
- 탑 꼭대기에 닿으면 클리어 시간과 남은 HP가 나옴.

### 꼭대기 클리어를 빨리 확인하려면

`-lab -reload`로 실행한 상태에서 `lab.rows`를 40, `lab.spacing`을 200 정도로 바꿔 저장하고 R. 통로가 짧아져 연사 몇 번이면 꼭대기의 체크무늬 결승선에 닿음. 일반 탑은 `tower.length`를 2~3으로 줄이고 게임을 다시 실행.

## 2. 통과 기준

- [ ] 1~2분 플레이가 "한 판 더"를 부른다.
- [ ] "탄창 높이 ÷ 적 간격" 비율의 적정 범위를 찾는다.

## 3. 비율 찾기 절차

1. `go run ./cmd/uplunge -lab -reload`
2. `data/tuning.json`의 `lab.spacing`을 바꿔 저장하고 R (실험 통로에서는 언제든 재시작됨). 탄창 높이(지금 약 134px)를 기준으로 아래 값을 차례로 시험.

| `lab.spacing` | 비율 (탄창 134px 기준) | 예상 |
|---|---|---|
| 64 | 약 2.1 | 적만 밟아도 쉽게 계속 오름 |
| 96 | 약 1.4 | 밟기 연쇄가 됨 (지금 기본값) |
| 128 | 약 1.05 | 경계: 정확히 쏘고 밟아야 이어짐 |
| 160 | 약 0.84 | 발판이 필요 |
| 192 | 약 0.7 | 적은 보조, 발판 위주 |

3. 각 간격에서 볼 것: 밟기 연쇄가 재미있게 이어지는가, 너무 쉬운가, 물에 쫓겨 급한가.
   - 화면의 비율은 "닿는 높이 ÷ 간격"으로, 바닥 출발(floor)과 밟은 뒤 출발(stomp) 두 가지를 보여 줌 (design.md 4장).
4. 결과로 "구역 1은 비율 ○○, 마지막 구역은 ○○" 같은 범위를 정함. 이 범위는 M2 조각 생성기와 구역 난이도의 기준이 됨 (design.md 4장).

## 4. 판정 항목 (🎮)

각 항목은 `data/tuning.json`에서 바로 바꿔 볼 수 있음. 현재 값은 모두 가설(🧪).

| 항목 | 현재 값 | 바꿀 곳 | 결과 |
|---|---|---|---|
| 추진력 (발사 한 발의 힘, 간격, 탄창) | 240px/s, 0.1초, 8발 | `gun.thrust`, `gun.fireInterval`, `gun.magazine` | |
| 점프 높이와 가변 점프 | 210px/s를 최대 0.2초 | `player.jumpSpeed`, `player.jumpHoldTime` | |
| 꼭대기 중력 절반 | 속도 80 미만에서 × 0.5 | `player.apexGravThreshold`, `player.apexGravMult` | |
| 좌우 이동과 미끄러짐 | 180px/s, 가속·감속 2000, 공중 × 0.65 | `player.runSpeed`, `player.runAccel`, `player.airAccelMult` | |
| 밟기 바운스 (적 윗면에서 출발, 늘어남과 흔들림 포함) | 280px/s를 0.2초, 흔들림 0.16초 2px | `player.stompSpeed`, `player.stompHoldTime`, `feel.stompShakeTime`, `feel.stompShakeScale` | |
| 넉백 (Downwell 측정값) | 좌우 180 × 각도, 위 190 | `player.knockbackX`, `player.knockbackY` | |
| 무적 시간 | 1.5초 | `player.invulnTime` | |
| 드릴 튕김 | 240px/s 한 번 | `player.drillBounce` | |
| 물 속도 | 30px/s (화면 밖이면 최대 3배) | `water.speed`, `water.maxMult` | |
| 물 안전망 튕김 | 400px/s | `water.bounce` | |
| 카메라 위치·미리 보기·추적 | 0.667, 0.2초, 1% | `camera.anchor`, `camera.lookahead`, `camera.remainPerSecond` | |
| 멈춤·흔들림·늘어남 | 밟기·총알 명중 1프레임, 드릴 0.05초 / 발사 0.2초 × 20px / (0.6, 1.4), (1.6, 0.4) | `feel` | |
| 드릴 손맛 (파편, 흔들림, 늘어남, 멈춤 길이) | 멈춤 0.05초, 흔들림 0.2초 × 15px, 파편 0.5초 | `feel.drillFreeze`, `feel.drillShake*`, `feel.debris*` | |
| 판정 박스 크기 | 12×20 (Celeste 환산 16×22도 시험) | `player.width`, `player.height` | |
| 기본 키 배치 | 위 조작 표 | 코드 (`internal/input`) | |
| 남길 블록 종류 | 드릴 / 무른 / 총알 전용 모두 있음 | `-chunk Blocks`, 탑의 DrillGate·SoftFloor | |
| 적 속도와 체력 | Floater 30px/s·2발, Spiker 제자리·3발 | `enemies` | |
| 탑 길이 (1~2분 맞는지) | 시작 조각 포함 24개 | `tower.length` | |

## 5. 결과 기록

- 날짜: 2026-10-11 (비율 실험)
- 통과 기준 판정:
  - [x] 비율 적정 범위를 찾는다 (아래)
  - [ ] 1~2분 플레이가 "한 판 더"를 부른다 (일반 탑으로 판정 대기)
- 비율 적정 범위 (실험 통로, Floater, 흔들림 16px):

| `lab.spacing` | 체감 |
|---|---|
| 64 | 적을 밟기는 매우 쉬움. 하지만 밟을 때 튕김 때문에 위의 적에게 맞는 경우가 생겨 난이도가 높음 |
| 96 | 최적으로 보임. 적 사이 거리가 괜찮고, 적만 밟고 오르는 길이라면 중상급자용 |
| 160 | 널널함. 바닥에서 밟기 좋고, 총을 쏘며 오르면서 적을 여유 있게 피하고 다음 적을 준비할 수 있음 |
| 180 | 바닥에서 점프하고 탄창을 다 써야 겨우 밟을 수 있음. 한 번 밟은 뒤에는 튕김 덕분에 계속 이어가기 쉬움. 고수용 |
| 192 | 바닥에서 적까지 닿긴 하나 한 번은 무조건 맞아야 함. 불가능한 루트 |

- 해석:
  - 넓은 쪽 한계는 "닿는 높이"와 맞음. 바닥 출발 187px (점프 꼭대기에서 탄창): 간격 180에서는 첫 적의 윗면이 평균 186px이라 겨우 닿고, 192에서는 평균 198px이라 대부분 못 닿음 (아래로 크게 흔들린 적에만 완벽하게 해야 닿음). 밟은 뒤 출발은 210px라서 180도 이어감.
  - 좁은 쪽은 닿기가 아니라 부딪힘 문제. 밟기 튕김(약 78px)이 위의 적까지 데려감.
  - 그래서 design.md 4장의 비율을 "닿는 높이 ÷ 간격"으로 다시 정의하고, 구역 기준 표를 만듦.
- 확정한 수치 (design.md에 ✅로 옮김): 핵심 비율의 재정의, 구역 난이도의 1차 기준.
- 다음에 고칠 것: 없음.
