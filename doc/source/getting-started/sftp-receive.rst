##############################
Réception d'un fichier en SFTP
##############################

Nous allons maintenant mettre en place des transferts SFTP avec la Gateway.
Pour ce faire, nous allons utiliser le serveur SSH local (OpenSSH) et le client
``sftp`` pour envoyer des fichiers vers la Gateway et en recevoir.

Pour pouvoir recevoir des fichiers, nous allons devoir ajouter un serveur SFTP à
la Gateway, puis créer un utilisateur, et une règle de transfert.

Création d'un serveur SFTP
==========================

Pour pouvoir recevoir des fichiers en SFTP avec la Gateway, nous allons
commencer par ajouter un serveur SFTP :

.. code-block:: shell-session

   # waarp-gateway server add --name "sftp_server" --protocol "sftp" --address "127.0.0.1:2223"
   The server sftp_server was successfully added.

Pour créer un serveur, nous devons préciser son nom, le protocole de ce serveur,
ainsi que des informations additionnelles pour paramétrer le serveur (ici
l'adresse écoutée et le port).

.. seealso::

   Plus d'options de configuration sont disponibles pour les serveurs sftp,
   notamment des options de sécurité.

   Le détail des options est disponible :any:`ici <proto-config-sftp>`

Nous devons maintenant ajouter une paire de clefs publique et privée pour le
nouveau serveur.
Nous pouvons utiliser ``ssh-keygen`` pour générer les clefs, puis nous les
ajoutons au serveur SFTP :

.. code-block:: shell-session

   # ssh-keygen -t rsa -b 4096 -f gateway-sftp -N "" -C "Waarp Gateway SFTP server"
   Generating public/private rsa key pair.
   Your identification has been saved in gateway-sftp.
   Your public key has been saved in gateway-sftp.pub.
   The key fingerprint is:
   SHA256:9G/DHpBTvR7V093tNKPekc8MbmkqIQA16WHLkO8RdmU Waarp Gateway SFTP server
   The key's randomart image is:
   +---[RSA 4096]----+
   |     oo. .E      |
   |    + *...   .  *|
   |     O =.   . .+O|
   |      B. . o  .==|
   |     . oS =  .+o.|
   |      . . .=.o.*o|
   |         . .*.*.+|
   |          .o *   |
   |           .o    |
   +----[SHA256]-----+

   # waarp-gateway server cert "sftp_server" add --name "sftp_hostkey" --private_key "./gateway-sftp" --public_key "./gateway-sftp.pub"
   The certificate sftp_server was successfully added.

Le serveur SFTP est maintenant créé mais n'est pas actif. Comme la Gateway doit
écouter un nouveau port, il est nécessaire de la redémarrer :

.. code-block:: shell-session

   # systemctl restart waarp-gatewayd
   # systemctl status waarp-gatewayd
   * waarp-gatewayd.service - Waarp Gateway server
      Loaded: loaded (/usr/lib/systemd/system/waarp-gatewayd.service; disabled; vendor preset: disabled)
      Active: active (running) since Thu 2020-08-27 08:52:23 UTC; 5s ago
    Main PID: 20584 (waarp-gatewayd)
       Tasks: 6 (limit: 2850)
      Memory: 3.3M
      CGroup: /system.slice/waarp-gatewayd.service
              └─20584 /usr/bin/waarp-gatewayd server -c /etc/waarp-gateway/waarp-gatewayd.ini

Si tout s'est bien passé, la commande ``status`` devrait lister le nouveau serveur :

.. code-block:: shell-session

   # waarp-gateway -a "http://admin:admin_password@127.0.0.1:8080" status
   Waarp-Gateway services:
   [Active]  Admin
   [Active]  Controller
   [Active]  Database
   [Active]  sftp_server

Création d'un utilisateur
=========================

Pour pouvoir se connecter au serveur, nous devons maintenant créer un
utilisateur. Cela se fait en créant un "compte local" dans la Gateway.
Cet utilisateur aura ``myuser`` comme login et ``mypassword`` comme mot de
passe :

.. code-block:: shell-session

   # waarp-gateway account local "sftp_server" add  --login "myuser" --password "mypassword"
   The account myuser was successfully added.

Nous pouvons essayer de nous connecter pour tester le paramétrage (entrez le mot
de passe quand celui-ci est demandé) :

.. code-block:: shell-session

   # sftp -P 2223 myuser@localhost
   The authenticity of host '[localhost]:2223 ([127.0.0.1]:2223)' can't be established.
   The authenticity of host '[localhost]:2223 ([127.0.0.1]:2223)' can't be established.
   RSA key fingerprint is SHA256:9G/DHpBTvR7V093tNKPekc8MbmkqIQA16WHLkO8RdmU.
   Are you sure you want to continue connecting (yes/no/[fingerprint])? yes
   Warning: Permanently added '[localhost]:2223' (RSA) to the list of known hosts.
   myuser@localhost's password: 
   Connected to myuser@localhost.
   sftp> quit

.. note::

   La demande de validation de la clef du serveur n'est demandée qu'une seule
   fois.

   Pour calculer l'empreinte de la clef que nous avons généré ci-dessus, la
   commande ``ssh-keygen -l -E sha256 -f gateway-sftp.pub`` peut être utilisée. L'empreinte
   générée par la commande doit correspondre à celle transmise par le serveur.


L'utilisateur est créé. Pour pouvoir faire un transfert, nous devons maintenant
créer une :term:`règle` de transfert

Ajout d'un règle
================

Les règles de transfert permettent de définir toutes les modalités liées à un
transfert : le sens du transfert, les dossiers utilisés comme source et
destination du fichier, les chaînes de traitement a exécuter avant ou après le
transfert et en cas d'erreur.

Pour Waarp Gateway, tous les transferts doivent être associés à une règle.
Cependant les clients ne peuvent pas fournir l'identifiant de la règle à
utiliser (le protocole SFTP ne le supporte pas). Waarp Gateway utilise donc le
chemin utilisé par le client. Quand celui-ci lit ou écrit un fichier, le dossier
dans lequel ce fichier est situé est comparé aux chemins des règles (propriété
``path``) pour déterminer la règle à utiliser. Si aucune règle n'est trouvée, le
transfert est refusé.

Ici, nous voulons envoyer un fichier à la Gateway. La règle aura donc le sens
``receive`` (« réception ») : le sens des règles est toujours à prendre du point
de vu de la Gateway (si on envoi un fichier à la Gateway, celle-ci le *reçoit*).

Assemblons tout dans une commande pour créer la règle :

.. code-block:: shell-session

   # waarp-gateway rule add --name "sftp_recv" --direction "receive" --path "sftp_recv"
   The rule sftp_recv was successfully added.

Premier transfert
=================

Maintenant que nous avons un serveur, un utilisateur et une règle, nous pouvons
effectuer un transfert. Créons d'abord un fichier à transférer et envoyons le à
la gateway :

.. code-block:: shell-session

   # echo "content of the file" > test.txt

   $ sftp -P 2223 myuser@localhost
   myuser@localhost's password: 
   Connected to myuser@localhost.
   sftp> put test.txt sftp_recv/test01.txt
   Uploading test.txt to /sftp_recv/test01.txt
   test.txt                                                                                              100%   20     5.7KB/s   00:00    
   sftp> quit

Après avoir établi une connexion avec la Gateway, nous avons déposé un fichier
avec la commande ``put`` dans le dossier ``sftp_recv`` que nous avons défini
ci-dessus comme le ``path`` de la règle ``sftp_recv``.

Nous pouvons vérifier que le transfert s'est bien passé dans l'historique des
transferts de la Gateway :

.. code-block:: shell-session

   $ waarp-gateway history list
   History:
   * Transfer 1 (as server) [DONE]
       Way:             receive
       Protocol:        sftp
       Rule:            sftp_recv
       Requester:       myuser
       Requested:       sftp_server
       Local filepath:  /etc/waarp-gateway/in/test01.txt
       Remote filepath: /test01.txt
       Start date:      2020-08-27T10:10:05Z
       End date:        2020-08-27T10:10:05Z
   
Le fichier disponible est maintenant dans le dossier ``in`` de la Gateway.
Comme nous n'avons pas spécifié de dossier spécifique dans la règle, c'est le
dossier par défaut du service qui est utilisé :

.. code-block:: shell-session

   # ls -l /var/lib/waarp-gateway/in/
   total 4
   -rw-------. 1 waarp waarp, 20 Aug 27 10:10 test01.txt

.. seealso::
   
   Plus d'informations sur la :any:`gestion des dossiers <gestion_dossiers>`.


