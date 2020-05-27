===================
Modifier un serveur
===================

.. program:: waarp-gateway server update

.. describe:: waarp-gateway <ADDR> server update <SERVER>

Remplace les attributs du serveur donné en paramètre par ceux fournis.
Les attributs omis resteront inchangés.

.. warning:: Les dossiers du serveur ne peuvent pas être modifiés individuellement.
   Pour modifier un des chemins, tous les autres doivent également être renseignés,
   sinon les anciennes valeurs seront perdues.

.. option:: -n <NAME>, --name=<NAME>

   Le nom du serveur. Doit être unique.

.. option:: -p <PROTO>, --protocol=<PROTO>

   Le protocole utilisé par le serveur.

.. option:: -r <ROOT>, --root=<ROOT>

   Le dossier racine du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine de la *gateway* renseignée
   dans le fichier de configuration.

.. option:: -i <IN_DIR>, --in=<IN_DIR>

   Le dossier de réception du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -o <OUT_DIR>, --out=<OUT_DIR>

   Le dossier d'envoi du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.

.. option:: -w <WORK_DIR>, --work=<WORK_DIR>

   Le dossier temporaire du serveur. Peut être un chemin relatif ou absolu. Si
   le chemin est relatif, il sera relatif à la racine du serveur.


.. option:: -c <CONF>, --config=<CONF>

   La configuration protocolaire du serveur.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 server update serveur_sftp -n server_sftp_new -r /sftp/root_new -i in -o out -w work -p sftp -c '{"address": "localhost", "port": 80}'
