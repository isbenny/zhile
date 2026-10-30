# Music DL · Midnight

Direction: Spotify-inspired dark music interface, retaining Music DL branding.

Tokens: page #121212, surfaces #1e1e1e / #242424, primary #1ed760, main text #ffffff, secondary #b3b3b3, borders #303030. System sans-serif. Rounded search tabs and circular play buttons. 16px mobile gutters, 44px touch targets, open song rows.

Reference: mobile-dark-concept.png. The earlier light concept was superseded by the user.

Functional deviations from the concept: real source lists and results remain dynamic; all original secondary song actions remain available through More; existing APlayer controls are retained; mobile sources initially collapse as in the original application.

Android identity: io.github.isbenny.musicdl, separate from upstream so both versions can coexist. The private signing key is stored outside the repository and supplied through GitHub encrypted secrets. Never commit signing material.
