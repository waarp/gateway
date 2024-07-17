==============================
Ajouter un compte à un serveur
==============================

.. program:: waarp-gateway account local add

Attache un nouveau compte au serveur donné à partir des informations renseignées.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" add

**Options**

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un serveur donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

.. option:: -i <IP_ADDRESS>, --ip-address <IP_ADDRESS>

   Restreint le compte à une adresse IP spécifique. Peut être répété pour
   restreindre le compte à plusieurs adresses. En l'absence d'adresse, le compte
   ne sera pas restreint à une adresse particulière.

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' add -l 'tata' -p 'password' -i '1.2.3.4' -i '5.6.7.8'
