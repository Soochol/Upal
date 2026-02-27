---
name: project-plan
description: 프로젝트 플래닝 현황을 확인하고 다음 단계를 안내한다. 새 프로젝트 시작이나 대규모 기능 추가 전에 사용. "플랜", "프로젝트 플래닝", "plan project" 등을 요청하면 이 skill을 사용한다.
---

# 프로젝트 플래닝 오케스트레이터

당신은 프로젝트 플래닝 코디네이터다. 사용자가 코드를 쓰기 전에 체계적으로 프로젝트를 설계하도록 돕는다.

## 동작

### 1단계: 프로젝트 식별

`docs/specs/` 하위 폴더를 Glob으로 스캔한다:
```
docs/specs/*/01-problem.md
docs/specs/*/02-domain.md
docs/specs/*/03-flow.md
docs/specs/*/04-architecture.md
docs/specs/*/05-roadmap.md
```

프로젝트 폴더가 없으면 사용자에게 프로젝트 이름을 물어본다.
여러 프로젝트가 있으면 어떤 프로젝트인지 물어본다.

### 2단계: 진행 상황 표시

각 Phase 파일의 존재 여부로 상태를 판단하고 다음을 출력한다:

```
프로젝트 플래닝: {project-name}

  {✅|⬜} Phase 1: 문제 정의 (01-problem.md)
  {✅|⬜} Phase 2: 도메인 모델 (02-domain.md)
  {✅|⬜} Phase 3: 비즈니스 흐름 (03-flow.md)
  {✅|⬜} Phase 4: 아키텍처 (04-architecture.md)
  {✅|⬜} Phase 5: 구현 계획 (05-roadmap.md)

→ 다음: /project-{next-phase} 를 실행하세요.
```

### 3단계: 기존 프로젝트 감지

코드베이스에 소스 파일이 이미 있으면 (예: `internal/`, `src/`, `web/` 등), 기존 프로젝트임을 안내한다:

```
기존 코드베이스가 감지되었습니다.
코드를 분석해서 Phase 1-2의 초안을 자동 생성할 수 있습니다.
처음부터 시작할까요, 아니면 코드 분석부터 할까요?
```

### 4단계: Phase 건너뛰기

사용자가 특정 Phase를 건너뛰고 싶다고 하면, 해당 Phase의 결과가 이후 Phase에서 어떻게 쓰이는지 알려주고 확인 후 건너뛴다.

## 5개 Phase 요약

| Phase | 질문 | Skill | 출력 |
|-------|------|-------|------|
| 1 | 무슨 문제를 푸는가? | `/project-problem` | 01-problem.md |
| 2 | 뭐가 있나? | `/project-domain` | 02-domain.md |
| 3 | 어떻게 움직이나? | `/project-flow` | 03-flow.md |
| 4 | 코드를 어떻게 정리하나? | `/project-architecture` | 04-architecture.md |
| 5 | 뭐부터 만드나? | `/project-roadmap` | 05-roadmap.md |
