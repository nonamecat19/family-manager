import {View} from 'react-native';
import {Input} from "@ui-kitten/components";
import {useState} from "react";

export default function HomeScreen() {
    const [value, setValue] = useState('');

    return (
        <View>
            <Input
                placeholder='Place your Text'
                value={value}
                onChangeText={nextValue => setValue(nextValue)}
            />
        </View>
    );
}
