<!-- 
<template>
     <div class="grid">
        <div class="card col-6">
        <CustomTriStateCheckBox v-model="checkboxState"></CustomTriStateCheckBox>
        <tracs-database-systems-transmitter-details></tracs-database-systems-transmitter-details>
    </div> 
    
        
    </div>
</template>

<script setup>
const checkboxState = ref(true);

</script> -->
<template>
    <!-- <div class="grid p-fluid">
        <div class="p-col-12 p-md-6 login-container">
            <h2>Admin Login</h2>
            <form @submit.prevent="login" class="login-form">
            <div class="form-group">
            <label for="username">Username / UserID:</label>
            <input type="text" id="username" v-model="credentials.username">
            </div>
            <div class="form-group">
            <label for="password">Password:</label>
            <input type="password" id="password" v-model="credentials.password">
            </div>
            <div class="form-group actions">
            <button type="submit">Login</button>
            <a href="#" @click="forgotPassword">Forgot Password?</a>
            </div>
        </form>
        </div>
        <div class="p-col-12 p-md-6 login-container">
            <h2>User Login</h2>
            <form @submit.prevent="login" class="login-form">
            <div class="form-group">
            <label for="username">Username / UserID:</label>
            <input type="text" id="username" v-model="credentials.username">
            </div>
            <div class="form-group">
            <label for="password">Password:</label>
            <input type="password" id="password" v-model="credentials.password">
            </div>
            <div class="form-group actions">
            <button type="submit">Login</button>
            <a href="#" @click="forgotPassword">Forgot Password?</a>
            </div>
        </form>
        </div>
    </div> -->
    <UserAdministration-login></UserAdministration-login>
    <!-- <Toast ref="toastMessage"></Toast> -->
</template>

  
<script setup>
import { ref } from 'vue';



</script>

  
