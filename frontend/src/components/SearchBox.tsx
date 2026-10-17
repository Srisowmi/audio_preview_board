interface Props {
  value: string;
  onChange: (value: string) => void;
}

export function SearchBox({ value, onChange }: Props) {
  return (
    <form role="search" className="searchbox" onSubmit={(e) => e.preventDefault()}>
      <label htmlFor="search-input" className="visually-hidden">
        Search for songs or artists
      </label>
      <input
        id="search-input"
        type="search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="Search songs or artists, e.g. coldplay"
        autoComplete="off"
        autoFocus
      />
    </form>
  );
}
