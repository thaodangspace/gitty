import { useAtom } from 'jotai';
import { selectedRepositoryAtom, selectedFilesAtom } from '@/store/atoms';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '@/lib/api-client';
import { Button } from '@/components/ui/button';
import { hljs, getLanguageFromPath } from '@/lib/highlight';
import { 
    File, 
    Loader2, 
    AlertCircle, 
    Download, 
    Edit3,
    Eye,
    FileText,
    Image as ImageIcon,
    Video,
    Music,
    Archive
} from 'lucide-react';

const getFileIcon = (fileName: string) => {
    const ext = fileName.split('.').pop()?.toLowerCase();
    
    switch (ext) {
        case 'jpg':
        case 'jpeg':
        case 'png':
        case 'gif':
        case 'svg':
        case 'webp':
            return <ImageIcon className="h-4 w-4" />;
        case 'mp4':
        case 'avi':
        case 'mov':
        case 'webm':
            return <Video className="h-4 w-4" />;
        case 'mp3':
        case 'wav':
        case 'flac':
        case 'ogg':
            return <Music className="h-4 w-4" />;
        case 'zip':
        case 'tar':
        case 'gz':
        case 'rar':
        case '7z':
            return <Archive className="h-4 w-4" />;
        default:
            return <FileText className="h-4 w-4" />;
    }
};

const isTextFile = (fileName: string): boolean => {
    const textExtensions = [
        'txt', 'md', 'js', 'ts', 'jsx', 'tsx', 'json', 'xml', 'html', 'css', 'scss', 'sass', 'less',
        'py', 'rb', 'php', 'java', 'c', 'cpp', 'h', 'hpp', 'cs', 'go', 'rs', 'swift', 'kt',
        'yml', 'yaml', 'toml', 'ini', 'cfg', 'conf', 'sh', 'bash', 'ps1', 'bat', 'cmd',
        'sql', 'log', 'gitignore', 'dockerfile', 'makefile', 'readme', 'license', 'changelog'
    ];
    
    const ext = fileName.split('.').pop()?.toLowerCase();
    return textExtensions.includes(ext || '') || fileName.toLowerCase().includes('readme') || 
           fileName.toLowerCase().includes('license') || fileName.toLowerCase().includes('changelog');
};

const isImageFile = (fileName: string): boolean => {
    const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp', 'ico'];
    const ext = fileName.split('.').pop()?.toLowerCase();
    return imageExtensions.includes(ext || '');
};


export default function FileViewer() {
    const [currentRepository] = useAtom(selectedRepositoryAtom);
    const [selectedFiles] = useAtom(selectedFilesAtom);

    const selectedFile = selectedFiles[0]; // For now, just show the first selected file

    const { data: fileContent, isLoading, error } = useQuery({
        queryKey: ['file-content', currentRepository?.id, selectedFile],
        queryFn: () => apiClient.getFileContent(currentRepository!.id, selectedFile),
        enabled: !!currentRepository?.id && !!selectedFile && isTextFile(selectedFile),
    });

    if (!currentRepository) {
        return (
            <div className="h-full flex items-center justify-center text-muted-foreground">
                <div className="text-center">
                    <File className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>No repository selected</p>
                </div>
            </div>
        );
    }

    if (!selectedFile) {
        return (
            <div className="h-full flex items-center justify-center text-muted-foreground">
                <div className="text-center">
                    <File className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>Select a file to view its contents</p>
                </div>
            </div>
        );
    }

    const fileName = selectedFile.split('/').pop() || selectedFile;
    const isText = isTextFile(fileName);
    const isImage = isImageFile(fileName);

    const renderFileHeader = () => (
        <div className="border-b p-3">
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                    {getFileIcon(fileName)}
                    <span className="font-medium">{fileName}</span>
                </div>
                <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm">
                        <Eye className="h-4 w-4 mr-1" />
                        View
                    </Button>
                    {isText && (
                        <Button variant="ghost" size="sm">
                            <Edit3 className="h-4 w-4 mr-1" />
                            Edit
                        </Button>
                    )}
                    <Button variant="ghost" size="sm">
                        <Download className="h-4 w-4 mr-1" />
                        Download
                    </Button>
                </div>
            </div>
            <div className="mt-1 text-sm text-muted-foreground">
                Path: {selectedFile}
            </div>
        </div>
    );

    const renderTextContent = () => {
        if (isLoading) {
            return (
                <div className="flex items-center justify-center h-24">
                    <div className="flex items-center gap-2 text-muted-foreground">
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span>Loading file content...</span>
                    </div>
                </div>
            );
        }

        if (error) {
            return (
                <div className="flex items-center justify-center h-24">
                    <div className="flex items-center gap-2 text-red-500">
                        <AlertCircle className="h-4 w-4" />
                        <span>Failed to load file content</span>
                    </div>
                </div>
            );
        }

        const language = getLanguageFromPath(fileName);
        const highlighted = fileContent
            ? language && hljs.getLanguage(language)
                ? hljs.highlight(fileContent, { language, ignoreIllegals: true }).value
                : hljs.highlightAuto(fileContent).value
            : '';

        return (
            <div className="p-3">
                <pre
                    className="hljs whitespace-pre-wrap text-sm font-mono bg-muted/30 p-3 rounded overflow-auto"
                    dangerouslySetInnerHTML={{ __html: highlighted }}
                />
            </div>
        );
    };

    const renderImageContent = () => (
        <div className="p-3 flex items-center justify-center">
            <img 
                src={`${import.meta.env.VITE_API_BASE}/repos/${currentRepository.id}/files/${selectedFile}`}
                alt={fileName}
                className="max-w-full max-h-96 object-contain border rounded"
                onError={(e) => {
                    const target = e.target as HTMLImageElement;
                    target.style.display = 'none';
                    target.nextElementSibling?.classList.remove('hidden');
                }}
            />
            <div className="hidden text-muted-foreground">
                <AlertCircle className="h-8 w-8 mx-auto mb-2" />
                <p>Unable to display image</p>
            </div>
        </div>
    );

    const renderBinaryContent = () => (
        <div className="p-3 flex items-center justify-center text-muted-foreground">
            <div className="text-center">
                {getFileIcon(fileName)}
                <p className="mt-2">Binary file - cannot display content</p>
                <p className="text-xs mt-1">Use the download button to save the file</p>
            </div>
        </div>
    );

    return (
        <div className="h-full flex flex-col">
            {renderFileHeader()}
            <div className="flex-1 overflow-auto">
                {isText && renderTextContent()}
                {isImage && renderImageContent()}
                {!isText && !isImage && renderBinaryContent()}
            </div>
        </div>
    );
}