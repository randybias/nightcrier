## MODIFIED Requirements

### Requirement: Markdown Report Generation
The runner SHALL generate a summary report in Markdown format at the end of each investigation with proper formatting for HTML conversion.

#### Scenario: Successful investigation report
- **WHEN** the agent completes its task successfully (exit code 0)
- **THEN** a `report.md` file is generated in the incident workspace
- **AND** it contains the incident ID, timestamp, severity, cluster name, and namespace
- **AND** it contains a summary section with one-paragraph executive overview
- **AND** it contains a findings section with detailed analysis
- **AND** it contains a recommendations section with suggested next steps
- **AND** it contains a metadata section with agent version, duration, and workspace path
- **AND** the markdown uses proper line break syntax (double-space line endings or double newlines) for metadata fields to ensure correct HTML rendering

#### Scenario: Failed investigation report
- **WHEN** the agent exits with a non-zero exit code
- **THEN** a `report.md` file is generated
- **AND** it includes the failure status, exit code, and any captured error output
- **AND** it contains metadata about the failure

#### Scenario: Timeout investigation report
- **WHEN** the agent times out before completing
- **THEN** a `report.md` file is generated
- **AND** it includes the timeout status and duration
- **AND** it contains any partial output captured before timeout

### Requirement: Report Template Structure
The runner SHALL use a consistent template structure for all reports with required sections and ensure proper HTML rendering.

#### Scenario: Required sections present
- **WHEN** generating any report
- **THEN** the report MUST include a header section with metadata
- **AND** it MUST include a summary section
- **AND** it MUST include a findings section
- **AND** it MUST include a recommendations section
- **AND** it MUST include a metadata footer

#### Scenario: Template rendering with special characters
- **WHEN** agent output contains Markdown special characters
- **THEN** the template MUST escape these characters to prevent formatting issues
- **AND** the rendered report displays the content correctly

#### Scenario: Metadata field line breaks in HTML
- **WHEN** the markdown report contains metadata fields in the format `**Field**: value`
- **THEN** the HTML conversion SHALL ensure these fields render on separate lines
- **AND** the converted HTML contains `<br>` tags or separate `<p>` elements for each metadata field
- **AND** the visual appearance in a browser shows each metadata field on its own line

## ADDED Requirements

### Requirement: HTML Report Generation
The runner SHALL convert markdown investigation reports to HTML format with proper formatting and styling.

#### Scenario: HTML conversion from markdown
- **WHEN** an investigation completes and `investigation.md` is generated
- **THEN** an `investigation.html` file is created through markdown-to-HTML conversion
- **AND** the HTML file includes proper CSS styling for readability
- **AND** the HTML file is saved alongside the markdown file
- **AND** both files contain the same semantic content

#### Scenario: HTML formatting preservation
- **WHEN** converting markdown to HTML
- **THEN** line breaks indicated by markdown syntax (double-space line endings or double newlines) SHALL render as `<br>` tags or paragraph breaks in HTML
- **AND** code blocks SHALL render with `<pre><code>` tags
- **AND** inline code SHALL render with `<code>` tags
- **AND** headings SHALL render with appropriate `<h1>` through `<h6>` tags
- **AND** lists SHALL render with `<ul>` or `<ol>` tags

#### Scenario: Metadata section formatting in HTML
- **WHEN** the markdown contains a metadata section with fields like `**Incident ID**: value` followed by `**Cluster**: value`
- **THEN** the HTML conversion SHALL preprocess the markdown to ensure proper line breaks
- **AND** each metadata field SHALL appear on a separate line in the rendered HTML
- **AND** the visual appearance matches the structured intent of the markdown

#### Scenario: HTML styling and readability
- **WHEN** generating HTML reports
- **THEN** the HTML includes embedded CSS for styling
- **AND** the CSS provides appropriate spacing for paragraphs, headings, and lists
- **AND** code blocks have distinct background colors and monospace fonts
- **AND** the report is readable on both desktop and mobile browsers
- **AND** the HTML includes a header section with incident badge and title

### Requirement: Markdown Preprocessing for HTML
The runner SHALL preprocess markdown content before HTML conversion to ensure proper formatting.

#### Scenario: Metadata field preprocessing
- **WHEN** markdown content contains lines matching the pattern `**Label**:` followed by a value
- **THEN** the preprocessor SHALL add double-space line endings if not already present
- **AND** the preprocessing SHALL be idempotent (repeated preprocessing produces same result)
- **AND** the preprocessing SHALL not modify code blocks or inline code

#### Scenario: Code block preservation
- **WHEN** markdown content contains fenced code blocks (```)
- **THEN** the preprocessor SHALL NOT modify content within code blocks
- **AND** code blocks SHALL render correctly in the final HTML

#### Scenario: Performance requirements
- **WHEN** preprocessing markdown for HTML conversion
- **THEN** the preprocessing SHALL complete in less than 10 milliseconds for typical reports (< 50KB)
- **AND** the preprocessing SHALL not cause memory allocations exceeding 2x the input size

### Requirement: Agent Prompt Guidance
The runner SHALL provide markdown formatting guidance in the agent prompt to encourage proper output.

#### Scenario: Prompt includes formatting instructions
- **WHEN** the runner invokes the agent
- **THEN** the agent prompt SHALL include instructions about markdown formatting
- **AND** the instructions SHALL specify the use of double newlines between metadata fields
- **AND** the instructions SHALL specify proper markdown syntax for code blocks and lists

#### Scenario: Defensive implementation
- **WHEN** the agent produces markdown that doesn't follow formatting instructions
- **THEN** the markdown preprocessor SHALL correct common issues automatically
- **AND** the HTML output SHALL still render correctly
- **AND** the system SHALL not fail due to imperfect agent output
