# VN - Vulnerability Navigator Flowcharts

This document contains comprehensive flowcharts documenting the functionality and architecture of the VN CLI tool.

## 1. Main CLI Flow

```mermaid
flowchart TD
    A[User runs 'vn' command] --> B{Command specified?}
    
    B -->|No| C[Show help and available commands]
    B -->|Yes| D{Which command?}
    
    D -->|sqli| E[SQL Injection Testing]
    D -->|xss| F[XSS Testing]
    D -->|misconfig| G[Security Misconfiguration Testing]
    D -->|help| C
    D -->|Invalid| H[Show error and help]
    
    E --> I[Parse sqli flags and arguments]
    F --> J[Parse xss flags and arguments]
    G --> K[Parse misconfig flags and arguments]
    
    I --> L[Execute SQL Injection Scanner]
    J --> M[Execute XSS Scanner]
    K --> N[Execute Misconfiguration Scanner]
    
    L --> O[Display SQL Injection Results]
    M --> P[Display XSS Results]
    N --> Q[Display Misconfiguration Results]
    
    O --> R[Exit with appropriate code]
    P --> R
    Q --> R
    C --> R
    H --> S[Exit with error code]
    
    style E fill:#ff6b6b
    style F fill:#4ecdc4
    style G fill:#45b7d1
    style R fill:#96ceb4
    style S fill:#feca57
```

## 2. SQL Injection Testing Flow

```mermaid
flowchart TD
    A[vn sqli URL] --> B[Parse Command Arguments]
    
    B --> C[Extract Configuration]
    C --> D[Method: GET/POST]
    C --> E[Custom Headers]
    C --> F[Parameters to Test]
    C --> G[POST Data]
    C --> H[Timeout & Threads]
    
    D --> I[Create SQLi Scanner]
    E --> I
    F --> I
    G --> I
    H --> I
    
    I --> J[Initialize HTTP Client]
    J --> K[Load SQL Injection Payloads]
    
    K --> L{Parameters specified?}
    L -->|No| M[Auto-detect parameters from URL]
    L -->|Yes| N[Use specified parameters]
    
    M --> O[Extract query parameters]
    N --> O
    
    O --> P[Start Concurrent Testing]
    P --> Q[Create Worker Goroutines]
    
    Q --> R[For each parameter]
    R --> S[For each payload type]
    
    S --> T[Error-based payloads]
    S --> U[Boolean-based payloads]
    S --> V[Time-based payloads]
    S --> W[Union-based payloads]
    S --> X[NoSQL payloads]
    
    T --> Y[Send HTTP Request]
    U --> Y
    V --> Y
    W --> Y
    X --> Y
    
    Y --> Z{Response Analysis}
    
    Z -->|SQL Error Detected| AA[Mark as Vulnerable]
    Z -->|Boolean Logic Success| AA
    Z -->|Time Delay Detected| AA
    Z -->|Union Query Success| AA
    Z -->|NoSQL Error Found| AA
    Z -->|No Vulnerability| BB[Continue to next]
    
    AA --> CC[Record Result with Evidence]
    BB --> DD{More payloads?}
    CC --> DD
    
    DD -->|Yes| S
    DD -->|No| EE{More parameters?}
    
    EE -->|Yes| R
    EE -->|No| FF[Aggregate Results]
    
    FF --> GG[Sort by Risk Level]
    GG --> HH[Display Results]
    
    HH --> II{Vulnerabilities found?}
    II -->|Yes| JJ[Show vulnerability details]
    II -->|No| KK[Show success message]
    
    JJ --> LL[Exit with findings]
    KK --> MM[Exit clean]
    
    style AA fill:#ff6b6b
    style JJ fill:#ff6b6b
    style KK fill:#96ceb4
    style MM fill:#96ceb4
```

## 3. XSS Testing Flow

```mermaid
flowchart TD
    A[vn xss URL] --> B[Parse Command Arguments]
    
    B --> C[Extract Configuration]
    C --> D[Method: GET/POST]
    C --> E[Custom Headers]
    C --> F[Parameters to Test]
    C --> G[POST Data]
    C --> H[Timeout & Threads]
    
    D --> I[Create XSS Scanner]
    E --> I
    F --> I
    G --> I
    H --> I
    
    I --> J[Initialize HTTP Client]
    J --> K[Load XSS Payloads]
    
    K --> L{Parameters specified?}
    L -->|No| M[Auto-detect parameters from URL]
    L -->|Yes| N[Use specified parameters]
    
    M --> O[Extract query parameters]
    N --> O
    
    O --> P[Start Concurrent Testing]
    P --> Q[Create Worker Goroutines]
    
    Q --> R[For each parameter]
    R --> S[For each XSS payload type]
    
    S --> T[Reflected XSS payloads]
    S --> U[DOM-based XSS payloads]
    S --> V[Filter bypass payloads]
    S --> W[Event handler payloads]
    S --> X[Encoded payloads]
    
    T --> Y[Send HTTP Request]
    U --> Y
    V --> Y
    W --> Y
    X --> Y
    
    Y --> Z{Response Analysis}
    
    Z -->|Payload Reflected| AA[Check for proper encoding]
    Z -->|DOM Manipulation| BB[Analyze client-side execution]
    Z -->|Filter Bypassed| CC[Mark as vulnerable]
    Z -->|No Reflection| DD[Continue to next]
    
    AA --> EE{Properly encoded?}
    EE -->|No| FF[Mark as Vulnerable - Reflected XSS]
    EE -->|Yes| DD
    
    BB --> GG{Executable context?}
    GG -->|Yes| HH[Mark as Vulnerable - DOM XSS]
    GG -->|No| DD
    
    CC --> II[Mark as Vulnerable - Filter Bypass]
    
    FF --> JJ[Record Result with Evidence]
    HH --> JJ
    II --> JJ
    DD --> KK{More payloads?}
    JJ --> KK
    
    KK -->|Yes| S
    KK -->|No| LL{More parameters?}
    
    LL -->|Yes| R
    LL -->|No| MM[Aggregate Results]
    
    MM --> NN[Sort by Risk Level]
    NN --> OO[Display Results]
    
    OO --> PP{Vulnerabilities found?}
    PP -->|Yes| QQ[Show vulnerability details]
    PP -->|No| RR[Show success message]
    
    QQ --> SS[Exit with findings]
    RR --> TT[Exit clean]
    
    style FF fill:#ff6b6b
    style HH fill:#ff6b6b
    style II fill:#ff6b6b
    style QQ fill:#ff6b6b
    style RR fill:#96ceb4
    style TT fill:#96ceb4
```

## 4. Security Misconfiguration Testing Flow

```mermaid
flowchart TD
    A[vn misconfig URL] --> B[Parse Command Arguments]
    
    B --> C[Extract Configuration]
    C --> D[Method: GET/POST]
    C --> E[Custom Headers]
    C --> F[Test Categories]
    C --> G[Timeout & Threads]
    
    D --> H[Create Misconfig Scanner]
    E --> H
    F --> H
    G --> H
    
    H --> I[Initialize HTTP Client]
    I --> J{Test categories specified?}
    
    J -->|No| K[Run all test categories]
    J -->|Yes| L[Run specified categories]
    
    K --> M[Test Categories: files, headers, defaults, server]
    L --> M
    
    M --> N[Start Concurrent Testing]
    N --> O[Create Worker Goroutines]
    
    O --> P{Test Category}
    
    P -->|files| Q[Sensitive Files Testing]
    P -->|headers| R[Security Headers Testing]
    P -->|defaults| S[Default Credentials Testing]
    P -->|server| T[Server Configuration Testing]
    
    Q --> Q1[Test for exposed files]
    Q1 --> Q2[/.env, /config.php, /.git/, /backup/]
    Q2 --> Q3[/admin/, /.htaccess, /web.config]
    Q3 --> Q4[Check HTTP response codes]
    Q4 --> Q5{File accessible?}
    Q5 -->|Yes| Q6[Mark as High Risk]
    Q5 -->|No| U[Continue to next test]
    
    R --> R1[Test security headers]
    R1 --> R2[X-Frame-Options, CSP, HSTS]
    R2 --> R3[X-Content-Type-Options, etc.]
    R3 --> R4[Check response headers]
    R4 --> R5{Headers missing?}
    R5 -->|Yes| R6[Mark as Medium Risk]
    R5 -->|No| U
    
    S --> S1[Test default credentials]
    S1 --> S2[admin:admin, admin:password]
    S2 --> S3[root:root, test:test]
    S3 --> S4[Send authentication requests]
    S4 --> S5{Authentication successful?}
    S5 -->|Yes| S6[Mark as High Risk]
    S5 -->|No| U
    
    T --> T1[Test server configuration]
    T1 --> T2[HTTP methods, redirects]
    T2 --> T3[Information disclosure]
    T3 --> T4[Check server responses]
    T4 --> T5{Misconfiguration found?}
    T5 -->|Yes| T6[Mark appropriate risk level]
    T5 -->|No| U
    
    Q6 --> V[Record Result with Evidence]
    R6 --> V
    S6 --> V
    T6 --> V
    U --> W{More tests?}
    V --> W
    
    W -->|Yes| P
    W -->|No| X[Aggregate Results]
    
    X --> Y[Sort by Risk Level and Category]
    Y --> Z[Generate Summary Statistics]
    Z --> AA[Display Results by Category]
    
    AA --> BB{Misconfigurations found?}
    BB -->|Yes| CC[Show detailed findings]
    BB -->|No| DD[Show success message]
    
    CC --> EE[Show remediation advice]
    EE --> FF[Exit with findings]
    DD --> GG[Exit clean]
    
    style Q6 fill:#ff6b6b
    style R6 fill:#feca57
    style S6 fill:#ff6b6b
    style T6 fill:#ff6b6b,#feca57,#96ceb4
    style CC fill:#ff6b6b
    style DD fill:#96ceb4
    style FF fill:#ff6b6b
    style GG fill:#96ceb4
```

## 5. Scanner Architecture Overview

```mermaid
flowchart TD
    A[CLI Commands Layer] --> B[cmd/root.go]
    A --> C[cmd/sqli.go]
    A --> D[cmd/xss.go]
    A --> E[cmd/misconfig.go]
    
    B --> F[Cobra CLI Framework]
    C --> G[SQLi Command Handler]
    D --> H[XSS Command Handler]
    E --> I[Misconfig Command Handler]
    
    G --> J[internal/scanner/sqli.go]
    H --> K[internal/scanner/xss.go]
    I --> L[internal/scanner/misconfig.go]
    
    J --> M[SQLi Scanner Implementation]
    K --> N[XSS Scanner Implementation]
    L --> O[Misconfig Scanner Implementation]
    
    M --> P[HTTP Client]
    N --> P
    O --> P
    
    M --> Q[Payload Management]
    N --> Q
    O --> Q
    
    M --> R[Response Analysis]
    N --> R
    O --> R
    
    M --> S[Result Aggregation]
    N --> S
    O --> S
    
    P --> T[Target Web Application]
    
    Q --> U[Error-based Payloads]
    Q --> V[Boolean-based Payloads]
    Q --> W[Time-based Payloads]
    Q --> X[XSS Payloads]
    Q --> Y[Misconfig Test Cases]
    
    R --> Z[Pattern Matching]
    R --> AA[Response Time Analysis]
    R --> BB[HTTP Status Analysis]
    R --> CC[Header Analysis]
    
    S --> DD[Risk Level Assignment]
    S --> EE[Evidence Collection]
    S --> FF[Remediation Suggestions]
    
    DD --> GG[Console Output]
    EE --> GG
    FF --> GG
    
    GG --> HH[Colored Terminal Display]
    
    style A fill:#e1f5fe
    style J fill:#ff6b6b
    style K fill:#4ecdc4
    style L fill:#45b7d1
    style T fill:#feca57
    style HH fill:#96ceb4
```

## 6. Data Flow Architecture

```mermaid
flowchart LR
    A[User Input] --> B[Command Parser]
    B --> C[Configuration Object]
    
    C --> D[Scanner Factory]
    D --> E{Scanner Type}
    
    E -->|SQLi| F[SQLi Scanner]
    E -->|XSS| G[XSS Scanner]
    E -->|Misconfig| H[Misconfig Scanner]
    
    F --> I[HTTP Request Builder]
    G --> I
    H --> I
    
    I --> J[Payload Injection]
    J --> K[HTTP Client]
    K --> L[Target Application]
    
    L --> M[HTTP Response]
    M --> N[Response Analyzer]
    
    N --> O{Vulnerability Detected?}
    O -->|Yes| P[Create Vulnerability Record]
    O -->|No| Q[Continue Testing]
    
    P --> R[Evidence Collection]
    R --> S[Risk Assessment]
    S --> T[Result Object]
    
    Q --> U{More Tests?}
    U -->|Yes| J
    U -->|No| V[Finalize Results]
    
    T --> W[Results Aggregator]
    V --> W
    
    W --> X[Results Formatter]
    X --> Y[Console Output]
    
    style A fill:#e1f5fe
    style L fill:#feca57
    style P fill:#ff6b6b
    style Y fill:#96ceb4
```

## Usage Examples

### Basic Usage Flow
```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Scanner
    participant Target
    
    User->>CLI: vn sqli https://example.com?id=1
    CLI->>CLI: Parse arguments and flags
    CLI->>Scanner: Create SQLi scanner with config
    Scanner->>Scanner: Load payloads
    Scanner->>Target: Send test requests
    Target-->>Scanner: HTTP responses
    Scanner->>Scanner: Analyze responses
    Scanner->>CLI: Return results
    CLI->>User: Display formatted output
```

PT
```mermaid
sequenceDiagram
    participant Usuário
    participant CLI
    participant Scanner
    participant Alvo
    
    Usuário->>CLI: vn sqli https://exemplo.com?id=1
    CLI->>CLI: Parser de argumentos e parâmetros
    CLI->>Scanner: Cria um scanner SQLi com configuração
    Scanner->>Scanner: Carrega payloads
    Scanner->>Alvo: Envia requisições de teste
    Alvo-->>Scanner: Respostas HTTP
    Scanner->>Scanner: Análise das respostas
    Scanner->>CLI: Retorno dos resultaods
    CLI->>Usuário: Apresenta resultados formatados
```

### Concurrent Testing Flow
```mermaid
sequenceDiagram
    participant Main
    participant Worker1
    participant Worker2
    participant WorkerN
    participant Target
    
    Main->>Worker1: Start goroutine with payload set 1
    Main->>Worker2: Start goroutine with payload set 2
    Main->>WorkerN: Start goroutine with payload set N
    
    par Concurrent Testing
        Worker1->>Target: Test with payloads 1-10
        and
        Worker2->>Target: Test with payloads 11-20
        and
        WorkerN->>Target: Test with payloads N-M
    end
    
    Worker1-->>Main: Results batch 1
    Worker2-->>Main: Results batch 2
    WorkerN-->>Main: Results batch N
    
    Main->>Main: Aggregate all results
    Main->>Main: Sort and format output
```