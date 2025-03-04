.. _cluster_tutorial:

##################################
Fonctionnement en grappe (cluster)
##################################

Depuis sa version 0.5, Waarp-Gateway est capable de fonctionner en grappe. Cela
implique de diviser une même instance de *gateway* en plusieurs sous-instances
(appelées "noeuds" ou *"nodes"* en anglais) permettant de répartir la charge de
transfert entre plusieurs machines, tout en continuant d'apparaitre comme une
seule instance de l'extérieur.

L'avantage d'un tel fonctionnement est qu'il permet non-seulement de meilleures
performances quand la charge de transferts devient très importante, mais il
permet également d'éviter d'éventuelles interruptions de service dans le cas où
une des instances viendrait à tomber.

Limitations
===========

Il est cependant important de noter que, dans son implémentation actuelle, le
fonctionnement en grappe de *Waarp-Gateway* est soumis à plusieurs limitations
et contraintes, et que celles-ci doivent être prises en compte lors de la
décision d'installer *Waarp-Gateway* en grappe. Ces limitations sont :

1) Tous les *nodes* doivent **impérativement** partager le même système de
   fichiers, sans quoi la reprise de transfert en cas d'avarie sera impossible.
2) Pour la même raison, les *nodes* doivent **impérativement** partager la même
   base de données. Cela assure également que les *nodes* seront bien identiques
   entre elles.
3) Le fonctionnement est grappe **nécessite** la présence d'un répartiteur de
   charge en amont des *nodes*. Ce répartiteur est nécessaire pour que les *nodes*
   apparaissent comme une seule instance d'un point de vue extérieur. À l'heure
   actuelle, *Waarp* ne fournit pas ce répartiteur de charge, une solution
   tierce devra donc être utilisée.
4) Bien que le fonctionnement en grappe permette d'éviter d'éventuelles
   interruptions de service, il est cependant impossible d'effectuer une
   mise-à-jour de *Waarp-Gateway* sans interruption de service. Cela est dû au
   fait que, comme précisé au point n°2, les *nodes* doivent partager la même
   base de données, et doivent donc avoir la même version du schéma de la base
   de données.

Installation
============

L'installation en grappe est similaire au processus d'installation de plusieurs
instances classiques, à la différence près que les *nodes* partagent toutes
le même fichier de configuration, la même base de données, et le même système
de fichiers.

Pour notre exemple, reprenons l'instance décrite précédemment dans ce guide
de démarrage. Nous souhaitons maintenant dupliquer cette instance en 3 noeuds
afin de pouvoir assurer un meilleur service. Nous appellerons respectivement
ces noeuds *node1*, *node2* et *node3*.

Dans notre exemple, les 3 noeuds tournent sur la même machine, mais la procédure
est identique dans le cas où les noeuds ne sont pas sur la même machine.

Configuration
=============

Gateway
-------

Dans le dossier où se trouve le fichier de configuration principal de notre
instance de *gateway*, il nous faudra créer un fichier d'*override* de configuration
pour chacun des 3 noeuds. Ces fichiers d'*override* permettent à chaque noeud
d'écraser une partie de la configuration globale de l'instance avec des valeurs
qui seront spécifiques à chacun des 3 noeuds.

Dans notre cas, il nous faut remplacer l'adresse du serveur SFTP, ainsi que
l'adresse du serveur REST d'administration.

Pour rappel dans notre guide, le serveur SFTP de la gateway avait pour adresse
*127.0.0.1:2223*, et le serveur REST d'administration avait pour adresse
*127.0.0.1:8080*.

Nous allons donc créer 3 fichiers d'*override* dans le même dossier où se trouve
le fichier de configuration principal. Ces fichiers doivent être au format *.ini*,
et doivent impérativement porter le même nom que leur *node* respectif. Par
conséquent, nous appelleront donc nos fichiers respectivement *node1.ini*,
*node2.ini* et *node3.ini*.

Pour *node1*, nous souhaitons que le serveur SFTP utilise l'adresse 127.0.0.1:2001,
et que le serveur REST utilise l'adresse 127.0.0.1:8081.

Le fichier *node1.ini* ressemblera donc à ceci :

.. code-block:: ini

   [Address Indirection]

   IndirectAddress = 127.0.0.1:2223 -> 127.0.0.1:2001
   IndirectAddress = 127.0.0.1:8080 -> 127.0.0.1:8081

Pour *node2*, nous souhaitons que le serveur SFTP utilise l'adresse 127.0.0.1:2002,
et que le serveur REST utilise l'adresse 127.0.0.1:8082.

Le fichier *node2.ini* ressemblera donc à ceci :

.. code-block:: ini

   [Address Indirection]

   IndirectAddress = 127.0.0.1:2223 -> 127.0.0.1:2002
   IndirectAddress = 127.0.0.1:8080 -> 127.0.0.1:8082

En enfin, pour *node3*, nous souhaitons que le serveur SFTP utilise l'adresse
127.0.0.1:2003, et que le serveur REST utilise l'adresse 127.0.0.1:8083.

Nous auront donc le fichier *node3.ini* suivant :

.. code-block:: ini

   [Address Indirection]

   IndirectAddress = 127.0.0.1:2223 -> 127.0.0.1:2003
   IndirectAddress = 127.0.0.1:8080 -> 127.0.0.1:8083


Proxy
-----

Une fois les indirections d'adresse configurées sur les 3 noeuds, il ne reste
plus qu'à configurer le répartiteur de charge. La marche à suivre pour cela
dépendra du répartiteur choisi. *Waarp* ne fournit pas ce répartiteur de charge,
une solution tierce devra donc être utilisée (telle que Nginx, ou Apache).

Quelle que soit la solution choisie, celui-ci devra donc être configurer pour
rediriger les connexions SFTP entrantes sur 127.0.0.1:2223 vers les 3 adresses
SFTP de nos 3 noeuds (respectivement 127.0.0.1:2001, 127.0.0.1:2002 et
127.0.0.1:2003).

De même, les connexions REST entrantes sur 127.0.0.1:8080 devront être redirigées
vers les adresses REST de nos 3 noeuds (respectivement 127.0.0.1:8081,
127.0.0.1:8082 et 127.0.0.1:8083).

Une fois la configuration du proxy terminée, celui-ci peut être démarré.

Lancement
=========

La commande pour lancer un noeud est la même que pour lancer une instance
classique. Il suffit simplement d'y ajouter l'option `-i` ou ``--instance``
suivie du nom du *node*.

Dans notre exemple, il faudra donc lancer la commande 3 fois avec le nom
de chacun des 3 noeuds :

.. code-block:: shell-session

   systemctl start waarp-gatewayd -i node1
   systemctl start waarp-gatewayd -i node2
   systemctl start waarp-gatewayd -i node3

Nous devrions donc avoir maintenant 3 services waarp-gatewayd en cours d'exécution
qui écoutent chacun sur leurs ports respectifs.

Pour se connecter en SFTP vers la grappe, il suffit donc simplement de se connecter
à l'adresse SFTP du répartiteur de charge (127.0.0.1:2223). Cette connexion sera
ensuite redirigée vers un des 3 noeuds de la grappe pour être traitée. De même,
pour se connecter au serveur REST d'administration, il suffit simplement de se
connecter à l'adresse 127.0.0.1:8080.

Nous avons donc bien 3 instances de *gateway* qui, de l'extérieur, apparaissent
comme une seule instance de *gateway*.
