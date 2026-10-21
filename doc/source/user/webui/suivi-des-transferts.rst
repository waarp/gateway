.. _user-webui-transfers:

####################
Suivi des transferts
####################

Section *Supervision*, adresse ``/webui/transfers``. La consultation requiert
le droit de lecture sur les transferts ; toute action (mettre en pause,
reprendre, annuler, relancer, ajouter, pré-enregistrer) requiert le droit
d'écriture sur les transferts, comme l'API REST.

Un transfert est l'échange d'un fichier selon une :ref:`règle <user-add-rule>`,
avec un partenaire distant (transfert *client*, à l'initiative de la Gateway)
ou pour un partenaire qui se connecte à un serveur local (transfert
*serveur*). Tant qu'il peut encore s'exécuter, il est dans la table des
transferts ; terminé ou annulé, il passe à l'historique. L'écran montre les
deux ensemble.


Liste
=====

En tête de page, une rangée compacte donne le nombre de transferts de chaque
état : *Tous*, *En cours*, *En erreur*, *Planifiés*, *En pause*,
*Interrompus*, *Disponibles*, *Terminés*, *Annulés*. Un clic sur un état
n'affiche que ses transferts ; l'état choisi est souligné et en gras.

La barre d'outils recherche un transfert par une partie du nom de son
fichier, ou par son identifiant de suivi. Le bouton *Filtres* ouvre les
critères : règle, sens (envoi ou réception), partenaire ou serveur, compte,
protocole, période de début. Un filtre s'applique dès qu'il change ; le
nombre de filtres actifs s'affiche sur le bouton, *Réinitialiser* les
retire tous. Les nombres de la rangée des états suivent la recherche et les
filtres.

.. list-table::
   :header-rows: 1
   :widths: 20 80

   * - Colonne
     - Contenu
   * - Fichier
     - Le nom du fichier, lien vers la fiche du transfert ; dessous, son
       numéro et son protocole.
   * - État
     - L'état du transfert (voir ci-dessous) ; pour un transfert en erreur,
       la cause de l'erreur.
   * - Règle
     - La règle ; dessous, le sens (*Envoi*, *Réception*) et le rôle de la
       Gateway (*client*, *serveur*).
   * - Partenaire
     - Le partenaire distant, ou le serveur local d'un transfert serveur ;
       dessous, le compte.
   * - Progression
     - Une barre et le pourcentage quand la taille du fichier est connue,
       les octets transférés sinon ; dessous, la taille.
   * - Début
     - L'heure et la date de début ; pour un transfert planifié, l'heure
       prévue ; pour un transfert disponible, sa date limite.
   * - Fin
     - L'heure et la date de fin ; pour un transfert en erreur qui sera
       retenté, l'heure de la prochaine tentative.

Par défaut, les transferts les plus récents viennent en premier. Les
colonnes *Fichier*, *État*, *Règle*, *Début* et *Fin* se trient d'un clic
sur leur en-tête ; l'état se trie dans l'ordre de la rangée des états. Le
pied de liste indique les lignes affichées, permet de changer de page et de
choisir le nombre de lignes par page (10, 25, 50 ou 100). La recherche, les
filtres, l'état, le tri et la page restent dans l'adresse de la page et sont
conservés pour la session : revenir à l'écran retrouve la liste telle
qu'elle a été laissée.

Le menu « ⋯ » de chaque ligne mène à la fiche du transfert et propose les
actions que son état permet (voir *Actions*).


Actualisation
-------------

La rangée des états, le nombre de transferts et le tableau sont actualisés
toutes les 5 secondes, sans recharger la page, en gardant la recherche, les
filtres, l'état, le tri et la page. L'heure de la dernière lecture s'affiche
en haut de page (*Actualisé à …*) ; le bouton *Actualiser* lit sans attendre.
L'actualisation attend pendant une saisie dans la barre d'outils et tant
qu'un menu « ⋯ » est ouvert. Si une lecture échoue (session expirée,
Gateway injoignable), un message persistant donne l'heure de la dernière
lecture réussie ; l'écran réessaie au cycle suivant, sans s'arrêter.


États
=====

.. list-table::
   :header-rows: 1
   :widths: 18 18 64

   * - État
     - Forme
     - Signification
   * - Planifié
     - Anneau gris
     - Transfert client qui attend son heure de départ.
   * - Disponible
     - Anneau gris
     - Transfert serveur pré-enregistré, qui attend que le partenaire se
       connecte, jusqu'à sa date limite.
   * - En cours
     - Rond vert
     - Le transfert s'exécute.
   * - En pause
     - Anneau gris
     - Le transfert a été mis en pause par un utilisateur.
   * - Interrompu
     - Triangle orange
     - Le transfert a été interrompu de façon inattendue, par un arrêt de la
       Gateway par exemple.
   * - En erreur
     - Losange rouge
     - Le transfert a échoué. Il reste dans la table des transferts : il
       peut être retenté automatiquement, repris ou relancé.
   * - Terminé
     - Rond vert
     - Le transfert, ses traitements avant et après transfert compris, s'est
       terminé sans erreur.
   * - Annulé
     - Anneau gris
     - Le transfert a été annulé par un utilisateur.


Fiche d'un transfert
====================

Le nom du fichier est le titre de la fiche ; l'état et le numéro du transfert
sont affichés dessous. La fiche réunit :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Carte
     - Contenu
   * - Erreur
     - Pour un transfert en erreur : la cause, libellé lisible du code
       d'erreur, avec le code technique (``TeConnection``…) ; le message tel
       que la Gateway l'a conservé ; l'étape où le transfert s'est arrêté.
   * - Transfert
     - La règle et le sens, le partenaire ou le serveur, le compte, le
       client, le protocole, l'identifiant distant et l'identifiant de suivi.
   * - Fichier
     - Les noms à la source et à la destination, le chemin sur la Gateway et
       le chemin chez le partenaire, la taille.
   * - Déroulement
     - Le début et la fin, l'étape et la tâche en cours, la progression.
   * - Tentatives
     - Pour un transfert client en cours de vie : les tentatives restantes,
       la prochaine tentative, le délai et son facteur d'augmentation.
   * - Informations de transfert
     - Les informations attachées au transfert, clé et valeur. Les clés
       réservées de la Gateway (``__followID__``…) sont nommées par ce
       qu'elles sont, leur nom technique dessous.

Le code d'erreur de la Gateway regroupe des causes voisines : un fichier déjà
existant ou occupé chez un partenaire PeSIT donne par exemple *Accès refusé*.
Le message conserve souvent le détail donné par le partenaire.


Actions
=======

Les actions dépendent de l'état du transfert ; le serveur vérifie l'état à
chaque action. Une action que l'état ne permet pas affiche une page qui dit
pourquoi.

.. list-table::
   :header-rows: 1
   :widths: 22 78

   * - Action
     - Effet et états concernés
   * - Mettre en pause
     - Suspend un transfert en cours, ou un transfert client planifié avant
       son départ.
   * - Reprendre
     - Reprend un transfert client en pause, interrompu ou en erreur, là où
       il s'est arrêté. Un transfert serveur reprend quand le partenaire se
       reconnecte ; le transfert enfant d'un transfert synchrone se reprend
       par son parent.
   * - Annuler le transfert
     - Après une page de confirmation qui nomme le transfert, son fichier et
       son état : le transfert passe à l'historique, à l'état *Annulé* ; un
       transfert en cours est d'abord interrompu.
   * - Relancer
     - Crée un nouveau transfert avec la même règle, le même partenaire, le
       même compte, le même client et le même fichier, maintenant ou à une
       date choisie ; le transfert d'origine reste tel quel et la fiche du
       nouveau s'affiche. Réservé aux transferts client en pause,
       interrompus, en erreur, terminés ou annulés. Si la règle, le
       partenaire, le compte ou le client n'existe plus, la relance est
       refusée.

Quand la Gateway fonctionne en grappe, un transfert en cours sur un autre
nœud ne se met pas en pause et ne s'annule pas depuis celui-ci : la page le
dit, l'action se fait depuis l'interface du nœud qui l'exécute.


Ajouter un transfert
====================

Le bouton *Ajouter un transfert* ouvre la page *Nouveau transfert* : un
transfert client, planifié à l'heure de départ choisie (maintenant si elle
est vide).

.. list-table::
   :header-rows: 1
   :widths: 28 72

   * - Champ
     - Contenu
   * - Fichier
     - À l'envoi, le chemin du fichier, relatif au dossier d'envoi de la
       règle ; à la réception, le nom du fichier à demander.
   * - Nom à la destination
     - Facultatif ; vide, le même nom que le fichier.
   * - Règle
     - Les règles, groupées par sens : la règle fixe le sens du transfert.
   * - Partenaire et compte
     - Les comptes distants, groupés par partenaire.
   * - Client
     - Facultatif ; vide, le client du protocole du partenaire. Quand
       plusieurs clients existent pour ce protocole, il faut en choisir un.
   * - Départ
     - Facultatif, à l'heure de la Gateway ; vide, maintenant.
   * - Informations de transfert
     - Facultatives, une par ligne, sous la forme ``clé: valeur`` ; les
       traitements de la règle peuvent les lire.
   * - Tentatives en cas d'erreur
     - Facultatives : nombre de tentatives, premier délai en secondes,
       facteur d'augmentation ; vides, celles du client.


Pré-enregistrer un transfert
============================

Le menu « ⋯ » de l'en-tête propose *Pré-enregistrer un transfert* : un
transfert serveur, que le partenaire échangera en se connectant au serveur
local, jusqu'à une date limite. Le transfert apparaît à l'état *Disponible*.
La page demande le fichier, la règle, le serveur et le compte local avec
lequel le partenaire se connectera, et la date limite ; les informations de
transfert sont facultatives.


Ancienne page
=============

L'ancienne page de suivi des transferts reste accessible à son adresse,
``/webui/transfer_monitoring``, jusqu'à la version suivante ; la barre
latérale mène au nouvel écran.
