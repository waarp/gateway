.. _user-webui-home:

#######
Accueil
#######

Adresse ``/webui/home``, page sur laquelle on arrive après la connexion et
par le nom du produit en haut à gauche. Tout utilisateur authentifié y
accède.

La page présente les écrans de l'interface, groupés comme la barre latérale :
*Connexions*, *Traitements*, *Supervision* et *Administration*, une colonne
par section. Chaque écran est une tuile, avec son icône, son nom et une
phrase qui dit ce qu'il contient ; la tuile entière mène à l'écran. Seuls les
écrans que l'utilisateur a le droit de lire sont proposés, et une section
sans écran accessible n'apparaît pas : un utilisateur sans aucun droit ne
voit que le :doc:`tableau-de-bord` et l'écran :doc:`etat-des-services`.

Les écrans déjà refondus mènent à leur nouvelle page ; les autres mènent à
leur page historique, jusqu'à leur refonte.


Menu du compte
==============

Le nom de l'utilisateur connecté, en haut à droite, ouvre un menu :

.. list-table::
   :header-rows: 1
   :widths: 25 75

   * - Entrée
     - Contenu
   * - Mon compte
     - La page ``/webui/account`` : l'identifiant et l'instance, les droits
       de l'utilisateur par domaine (lecture, écriture, suppression, fixés
       par un administrateur dans *Utilisateurs*), et le changement de son
       mot de passe : le mot de passe actuel, puis le nouveau saisi deux
       fois. Le mot de passe est conservé haché.
   * - À propos
     - La version de la Gateway, sa date de compilation et sa révision, le
       rappel du support et des services professionnels de Waarp, la
       licence, et les liens utiles (téléchargement, documentation, forums,
       contact). Avec JavaScript, une fenêtre sur la page en cours, fermée
       par la croix, le bouton *Fermer*, la touche Échap ou un clic en
       dehors ; sans JavaScript, la page ``/webui/about`` au même contenu.
   * - Se déconnecter
     - Ferme la session.

Le bouton *Documentation* de l'en-tête ouvre la dernière documentation
publiée de Waarp Gateway. La version installée et sa révision sont aussi
rappelées au pied de la barre latérale.
