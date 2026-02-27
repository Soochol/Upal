# 비즈니스 로직 문서

코드를 읽지 않고 시스템의 동작을 이해하기 위한 자연어 비즈니스 로직 문서.
각 도메인의 핵심 규칙, 상태 흐름, 비즈니스 결정을 설명한다.

## 도메인

### [Session](session/)
콘텐츠 수집-분석-제작 파이프라인.

| 문서 | 설명 |
|------|------|
| [lifecycle.md](session/lifecycle.md) | Session/Run 상태 머신, 전환 규칙, 삭제 cascade |
| [content-collection.md](session/content-collection.md) | 소스 수집, LLM 분석, 워크플로우 매칭 |
| [production.md](session/production.md) | 워크플로우 병렬 실행, 발행, 자동 생성 |
