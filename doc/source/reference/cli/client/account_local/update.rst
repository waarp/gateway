========================
Modifier un compte local
========================

.. program:: waarp-gateway account local update

Remplace les attributs du compte par ceux renseignés.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" update "<LOGIN>"

**Options**

.. option:: -l <LOGIN>, --login=<LOGIN>

   Change le nom d'utilisateur du compte. Doit être unique pour un
   serveur donné.

.. option:: -p <PASS>, --password=<PASS>

   Change le mot de passe du compte.

.. option:: -i <IP_ADDRESS>, --ip-address <IP_ADDRESS>

   Restreint le compte à une adresse IP spécifique. Peut être répété pour
   restreindre le compte à plusieurs adresses. En l'absence d'adresse, le compte
   ne sera pas restreint à une adresse particulière. Pour enlever toutes les
   adresses existantes, utiliser la valeur ``none``.

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' update 'tata' -l 'tutu' -p 'password_new' -i '9.8.7.6'
