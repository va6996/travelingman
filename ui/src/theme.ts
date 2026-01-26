
import { extendTheme } from '@chakra-ui/react'

const theme = extendTheme({
    styles: {
        global: {
            'html, body': {
                fontFamily: "'Recoleta', serif",
                color: 'white',
                bg: '#3d5446',
            },
            '*': {
                fontFamily: "'Recoleta', serif !important",
            }
        },
    },
    fonts: {
        heading: "'Recoleta', serif",
        body: "'Recoleta', serif",
    },
    components: {
        Text: {
            baseStyle: {
                fontFamily: "'Recoleta', serif",
            }
        },
        Heading: {
            baseStyle: {
                fontFamily: "'Recoleta', serif",
            }
        },
        Button: {
            baseStyle: {
                fontFamily: "'Recoleta', serif",
            }
        },
        Input: {
            baseStyle: {
                field: {
                    fontFamily: "'Recoleta', serif",
                }
            }
        }
    }
})

export default theme
