##########################
Envoi d'un fichier en SFTP
##########################

.. todo::

   Make this page use the R66 protocol when ready

Nous allons maintenant mettre en place le second transfert : l'envoi d'un
fichier avec la Gateway vers un serveur SSH. Nous allons utiliser le serveur
OpenSSH installé sur le serveur comme serveur de destination.

Pour pouvoir envoyer des fichiers, nous allons devoir ajouter un partenaire SFTP
à la Gateway, puis créer un utilisateur, et une règle de transfert.

Configuration du serveur OpenSSH local
======================================

La configuration par défaut du serveur est suffisante. Nous allons cependant
créer un utilisateur, que nous utiliserons avec la Gateway.

OpenSSH utilise les utilisateurs UNIX standards :

.. code-block:: shell-session

   # adduser --create-home --user-group --password mypassword sftpuser



C'est suffisant, on va maintenant pouvoir se connecter au serveur ``localhost``
sur le port ``22`` (port par défaut), avec l'utilisateur ``sftpuser`` et le mot
de passe ``mypassword``.


Création d'un partenaire SFTP
=============================

Pour pouvoir envoyer des fichiers en SFTP avec la Gateway, nous allons
commencer par ajouter un partenaire SFTP :

.. code-block:: shell-session

   $ waarp-gateway partner add -n sftp_localhost -p sftp -a localhost:22 -c '{}'
   The partner sftp_localhost was successfully added.

Pour créer un partenaire, nous devons préciser son nom, le protocole de ce
serveur, ainsi que des informations additionnelles pour paramétrer le serveur
(ici l'adresse écoutée et le port).

.. seealso::

   Plus d'options de configuration sont disponibles pour les partenaires sftp,
   notamment des options de sécurité.

   Le détail des options est disponible :any:`ici <proto-config-sftp>`

Nous devons maintenant ajouter la clef publique du partenaire pour pouvoir
l'authentifier.

Comme nous utilisons le serveur SSH intégré à notre serveur, on peut récupérer
sa clef à l'enplacement par défaut : :file:`/etc/ssh/ssh_host_rsa_key.pub` :

.. code-block:: shell-session

   $ waarp-gateway partner cert sftp_localhost add -n sftp_localhost_cert -b /etc/ssh/ssh_host_rsa_key.pub
   The certificate sftp_localhost_cert was successfully added.


Création d'un utilisateur
-------------------------

Pour pouvoir se connecter au partenaire, nous devons maintenant créer un
utilisateur. Cela se fait en créant un "compte distant" dans la Gateway.
Cet utilisateur aura ``sftpuser`` comme login et ``mypassword`` comme mot de
passe (ceux définis plus tôt lors de la création de l'utilisateur système) :

.. code-block:: shell-session

   $ waarp-gateway account remote sftp_localhost add -l sftpuser -p mypassword
   The account sftpuser was successfully added.

L'utilisateur est maintenant créé. Pour pouvoir faire un transfert, nous devons
maintenant créer une :term:`règle` de transfert


Ajout d'un règle
----------------


Ici, nous voulons envoyer avec fichier à la Gateway. La règle aura donc le sens
``SEND`` (« envoi ») : le sens des règles est toujours à prendre du point
de vu de la Gateway (si on envoi un fichier à la Gateway, celle-ci le *reçoit*).

Le chemin doit être renseigné pour la règle : celui-ci est obligatoire. Il ne
sera pas utilisé pour déterminer la règle, puisque dans le cas d'un envoi depuis
la Gateway, la règle est donnée lors de la création du transfert. Nous
renseignerons ce chemin par convention avec le nom de la règle.

Assemblons tout dans une commande pour créer la règle :

.. code-block:: shell-session

   $ waarp-gateway rule add -n sftp_send -d SEND -p sftp_send
   The rule sftp_send was successfully added.


Premier transfert
-----------------

Maintenant que nous avons un partenaire, un utilisateur et une règle, nous
pouvons effectuer un transfert. Créons d'abord un fichier à transférer et
envoyons le avec la gateway :

.. code-block:: shell-session

   # echo "hello world!" > /var/lib/waarp-gateway/out/a-envoyer.txt

   $ transfer add -f a-envoyer.txt -w push -p sftp_localhost -l sftpuser -r sftp_send
   The transfer of file a-envoyer.txt was successfully added.

Après avoir établi une connexion avec la Gateway, nous avons déposé un fichier
avec la commande ``put`` dans le dossier ``sftp_recv`` que nous avons défini
ci-dessus comme le ``path`` de la règle ``sftp_recv``.

Nous pouvons vérifier que le transfert s'est bien passé dans l'historique des
transferts de la Gateway :

.. code-block:: shell-session

   $ waarp-gateway history list
   History:
   [...]
   ● Transfer 2 (as client) [DONE]
       Way:              SEND
       Protocol:         sftp
       Rule:             sftp_send
       Requester:        sftpuser
       Requested:        sftp_localhost
       Source file:      a-envoyer.txt
       Destination file: a-envoyer.txt
       Start date:       2020-09-17T17:27:44Z
       End date:         2020-09-17T17:27:45Z
   
Le fichier disponible est maintenant dans le dossier ``in`` de la Gateway.
Comme nous n'avons pas spécifié de dossier spécifique dans la règle, c'est le
dossier par défaut du service qui est utilisé :

.. code-block:: shell-session

   # ls -l /home/sftpuser/
   total 4
   -rw-rw-r--. 1 sftpuser sftpuser 13 Sep 17 17:27 a-envoyer.txt

.. seealso::
   
   Plus d'informations sur la gestion des dossiers.

.. todo:: Créer une page gestion des dossiers

