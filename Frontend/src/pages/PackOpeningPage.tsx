import { useParams } from 'react-router-dom';

export function PackOpeningPage() {
    const { setCode } = useParams<{ setCode: string }>();
}
