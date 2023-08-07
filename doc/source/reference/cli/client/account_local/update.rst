========================
Modifier un compte local
========================

.. program:: waarp-gateway account local update

Remplace les attributs du compte par ceux renseignés.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Change le nom d'utilisateur du compte. Doit être unique pour un
   serveur donné.

.. option:: -p <PASS>, --password=<PASS>

   Change le mot de passe du compte.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' account local 'serveur_sftp' update 'tata' -l 'tutu' -p 'password_new'
